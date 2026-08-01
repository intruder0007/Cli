package sdk

// Byte-level golden tests for the plugin wire protocol. Together with
// core/plugin/testdata/wire-generate.golden and wire-apply.golden, these
// pin the exact bytes exchanged between the host and a plugin so the
// wire contract (docs/architecture/plugin-protocol.md) can't drift
// silently. Regenerate with: go test ./sdk/go/sdk/ -run ServeWire -update

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var updateGolden = flag.Bool("update", false, "rewrite wire-protocol golden transcript files")

// lifecyclePlugin implements both SDK interfaces with fixed, deterministic
// responses so the emitted bytes can be pinned exactly.
type lifecyclePlugin struct{}

func (lifecyclePlugin) Generate(GenerateRequest) (GenerateResponse, error) {
	return GenerateResponse{
		FilesWritten: []string{"go.mod", "main.go"},
		NextSteps:    []string{"cd new-project && go run ."},
	}, nil
}

func (lifecyclePlugin) Apply(ApplyRequest) (ApplyResponse, error) {
	return ApplyResponse{
		FilesWritten:  []string{"README.md", "Makefile"},
		FilesModified: []string{".gitignore"},
		NextSteps:     []string{"git add -A"},
	}, nil
}

// templateOnlyPlugin implements Generate only, for the
// "does not implement apply" error contract.
type templateOnlyPlugin struct{}

func (templateOnlyPlugin) Generate(GenerateRequest) (GenerateResponse, error) {
	return GenerateResponse{}, errors.New("simulated failure")
}

// transcript records the byte-level wire exchange between a scripted
// host and serveWithIO, line-oriented with direction markers: "> " is
// host-to-plugin, "< " is plugin-to-host.
type transcript struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	writerP []byte // partial line pending from the writer side
	readerP []byte // partial line pending from the reader side
}

func newTranscript() *transcript { return &transcript{} }

func (t *transcript) writer(marker string, w io.Writer) io.Writer {
	return &recWriter{t: t, marker: marker, w: w}
}

func (t *transcript) reader(marker string, r io.Reader) io.Reader {
	return &recReader{t: t, marker: marker, r: r}
}

func (t *transcript) record(marker string, pending *[]byte, p []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	*pending = append(*pending, p...)
	for {
		i := bytes.IndexByte(*pending, '\n')
		if i < 0 {
			return
		}
		t.buf.WriteString(marker)
		t.buf.Write((*pending)[:i])
		t.buf.WriteByte('\n')
		*pending = (*pending)[i+1:]
	}
}

func (t *transcript) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}

type recWriter struct {
	t       *transcript
	marker  string
	w       io.Writer
	pending []byte
}

func (r *recWriter) Write(p []byte) (int, error) {
	r.t.record(r.marker, &r.pending, p)
	return r.w.Write(p)
}

type recReader struct {
	t       *transcript
	marker  string
	r       io.Reader
	pending []byte
}

func (r *recReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.t.record(r.marker, &r.pending, p[:n])
	}
	return n, err
}

// withServer runs serveWithIO over pipes with a scripted host on the
// other side. send/recv speak one request line at a time, exactly the
// host's call/response pattern; done closes the input and waits for the
// loop to exit (at plugin.shutdown or EOF).
func withServer(t *testing.T, plugin interface{}, manifest Manifest) (send func(string), recv func() string, done func(), tr *transcript) {
	t.Helper()
	hostInR, hostInW := io.Pipe()
	hostOutR, hostOutW := io.Pipe()
	tr = newTranscript()
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		serveWithIO(plugin, manifest, tr.reader("> ", hostInR), tr.writer("< ", hostOutW))
	}()
	br := bufio.NewReader(hostOutR)
	send = func(line string) {
		if _, err := hostInW.Write([]byte(line + "\n")); err != nil {
			t.Errorf("sending to server: %v", err)
		}
	}
	recv = func() string {
		b, err := br.ReadBytes('\n')
		if err != nil {
			t.Errorf("reading from server: %v", err)
		}
		return string(b)
	}
	done = func() {
		hostInW.Close()
		select {
		case <-serverDone:
		case <-time.After(5 * time.Second):
			t.Error("server did not exit after input closed")
		}
	}
	return send, recv, done, tr
}

func checkGolden(t *testing.T, tr *transcript, goldenPath string) {
	t.Helper()
	got := tr.String()
	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", goldenPath)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("wire transcript mismatch (golden %s):\n--- want ---\n%s\n--- got ---\n%s", goldenPath, want, got)
	}
}

// TestServeWireLifecycleTranscript pins the full documented lifecycle —
// initialize, generate, apply, shutdown — byte-for-byte. The scripted
// host lines are the exact bytes core/plugin's host emits (verified
// against core/plugin/testdata/wire-*.golden), so both sides of the
// wire are anchored to the same contract.
func TestServeWireLifecycleTranscript(t *testing.T) {
	manifest := validTemplateManifest()
	manifest.DisplayName = "Test Template"

	send, recv, done, tr := withServer(t, lifecyclePlugin{}, manifest)
	defer done()

	send(`{"jsonrpc":"2.0","id":1,"method":"plugin.initialize","params":{"protocolVersion":"1"}}`)
	recv()
	send(`{"jsonrpc":"2.0","id":2,"method":"plugin.generate","params":{"answers":{"framework":"rest-api","language":"go","projectType":"backend-service","theme":"default"},"projectName":"new-project","targetDir":"/abs/path/to/new-project"}}`)
	recv()
	send(`{"jsonrpc":"2.0","id":3,"method":"plugin.apply","params":{"answers":{"framework":"rest-api","language":"go","projectType":"backend-service","theme":"default"},"projectName":"new-project","targetDir":"/abs/path/to/new-project"}}`)
	recv()
	send(`{"jsonrpc":"2.0","id":4,"method":"plugin.shutdown","params":{}}`)
	recv()

	checkGolden(t, tr, filepath.Join("testdata", "serve-lifecycle.golden"))
}

// TestServeWireErrorResponses pins the JSON-RPC error contract for the
// errors whose messages the SDK itself produces. The -32700 (parse) and
// -32602 (invalid params) messages are encoding/json's own text and
// vary across Go versions, so those are checked structurally.
func TestServeWireErrorResponses(t *testing.T) {
	manifest := validTemplateManifest()

	send, recv, done, tr := withServer(t, templateOnlyPlugin{}, manifest)
	defer done()

	send(`{"jsonrpc":"2.0","id":7,"method":"plugin.bogus","params":{}}`)
	recv()
	send(`{"jsonrpc":"2.0","id":8,"method":"plugin.apply","params":{"targetDir":"/x","projectName":"y","answers":{}}}`)
	recv()
	send(`{"jsonrpc":"2.0","id":9,"method":"plugin.generate","params":{"targetDir":"/x","projectName":"y","answers":{}}}`)
	recv()

	checkGolden(t, tr, filepath.Join("testdata", "serve-errors.golden"))
}

func TestServeWireParseErrorIsStructural(t *testing.T) {
	manifest := validTemplateManifest()
	send, recv, done, _ := withServer(t, lifecyclePlugin{}, manifest)
	defer done()

	send(`this is not json`)
	var resp rpcResponse
	if err := json.Unmarshal([]byte(recv()), &resp); err != nil {
		t.Fatalf("response is not decodable JSON: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != -32700 || resp.Error.Message == "" {
		t.Errorf("got %+v, want -32700 with a message", resp)
	}
}

func TestServeWireInvalidParamsIsStructural(t *testing.T) {
	manifest := validTemplateManifest()
	send, recv, done, _ := withServer(t, lifecyclePlugin{}, manifest)
	defer done()

	send(`{"jsonrpc":"2.0","id":2,"method":"plugin.generate","params":{"targetDir":42}}`)
	var resp rpcResponse
	if err := json.Unmarshal([]byte(recv()), &resp); err != nil {
		t.Fatalf("response is not decodable JSON: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != -32602 || resp.Error.Message == "" {
		t.Errorf("got %+v, want -32602 with a message", resp)
	}
}
