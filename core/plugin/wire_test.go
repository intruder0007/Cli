package plugin

// Byte-level golden tests for the host side of the plugin wire
// protocol. Each test builds core/plugin/wiretest (a real plugin binary
// served by sdk/go), drives it through the exact lifecycle Host uses
// (initialize -> generate/apply -> shutdown, the same sequence
// Generate/Apply run), and pins every byte in both directions against
// a golden transcript. Together with sdk/go/sdk/testdata/*.golden these
// anchor both sides of the wire to the contract in
// docs/architecture/plugin-protocol.md.
// Regenerate with: go test ./core/plugin/ -run WireTranscript -update

import (
	"bufio"
	"bytes"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	sdk "github.com/intruder0007/Lumo/sdk/go/sdk"
)

var updateGolden = flag.Bool("update", false, "rewrite wire-protocol golden transcript files")

const templateManifestJSON = `{
  "protocolVersion": "1",
  "name": "wiretest",
  "version": "0.1.0",
  "kind": "template",
  "displayName": "Wire Test Template",
  "projectType": "backend-service",
  "language": "go",
  "framework": "rest-api",
  "entrypoint": "./wiretest"
}`

const capabilityManifestJSON = `{
  "protocolVersion": "1",
  "name": "wiretest",
  "version": "0.1.0",
  "kind": "capability",
  "displayName": "Wire Test Capability",
  "capabilityId": "wiretest-cap",
  "entrypoint": "./wiretest"
}`

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// transcript records the byte-level wire exchange between the host and
// the responder subprocess, line-oriented with direction markers: "> "
// is host-to-plugin, "< " is plugin-to-host.
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

// buildWireResponder compiles core/plugin/wiretest into dir and returns
// the executable path. It needs the Go toolchain at test time, exactly
// like tests/integration does.
func buildWireResponder(t *testing.T, dir string) string {
	t.Helper()
	exe := filepath.Join(dir, exeName("wiretest"))
	cmd := exec.Command("go", "build", "-o", exe, "github.com/intruder0007/Lumo/core/plugin/wiretest")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building wiretest responder: %v\n%s", err, out)
	}
	return exe
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// startRecordingSession is start() with recording pipes: the spawned
// responder's stdio passes through the transcript so every byte in both
// directions is captured.
func startRecordingSession(t *testing.T, entrypointPath string, tr *transcript) *session {
	t.Helper()
	cmd := exec.Command(entrypointPath)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("opening responder stdin: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("opening responder stdout: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting responder: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return &session{
		cmd:    cmd,
		stdin:  bufio.NewWriter(tr.writer("> ", stdinPipe)),
		stdout: bufio.NewReader(tr.reader("< ", stdoutPipe)),
	}
}

// wireRoundTrip drives a real responder through the same lifecycle
// Generate/Apply run — initialize, the named call, then shutdown via
// finish — recording the full transcript.
func wireRoundTrip(t *testing.T, manifestJSON, method string, params, result interface{}, golden string) *transcript {
	t.Helper()
	dir := t.TempDir()
	exe := buildWireResponder(t, dir)
	writeFile(t, filepath.Join(dir, "plugin.json"), manifestJSON)

	tr := newTranscript()
	s := startRecordingSession(t, exe, tr)

	h := NewHost()
	h.CallTimeout = 5 * time.Second
	h.ShutdownTimeout = 5 * time.Second

	if err := h.initialize(s, "wiretest", sdk.ProtocolVersion); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := s.call(h.callTimeout(), method, params, result); err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	h.finish(s)

	checkGolden(t, tr, golden)
	return tr
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

func TestGenerateWireTranscript(t *testing.T) {
	var out sdk.GenerateResponse
	wireRoundTrip(t, templateManifestJSON, "plugin.generate", sdk.GenerateRequest{
		TargetDir:   "/abs/path/to/new-project",
		ProjectName: "new-project",
		Answers: map[string]string{
			"theme": "default", "projectType": "backend-service", "language": "go", "framework": "rest-api",
		},
	}, &out, filepath.Join("testdata", "wire-generate.golden"))

	if len(out.FilesWritten) != 2 || out.FilesWritten[0] != "go.mod" || out.FilesWritten[1] != "main.go" {
		t.Errorf("FilesWritten = %v, want [go.mod main.go]", out.FilesWritten)
	}
	if len(out.NextSteps) != 1 || out.NextSteps[0] != "cd new-project && go run ." {
		t.Errorf("NextSteps = %v, want [cd new-project && go run .]", out.NextSteps)
	}
}

func TestApplyWireTranscript(t *testing.T) {
	var out sdk.ApplyResponse
	wireRoundTrip(t, capabilityManifestJSON, "plugin.apply", sdk.ApplyRequest{
		TargetDir:   "/abs/path/to/new-project",
		ProjectName: "new-project",
		Answers: map[string]string{
			"theme": "default", "projectType": "backend-service", "language": "go", "framework": "rest-api",
		},
	}, &out, filepath.Join("testdata", "wire-apply.golden"))

	if len(out.FilesWritten) != 2 || out.FilesWritten[0] != "README.md" || out.FilesWritten[1] != "Makefile" {
		t.Errorf("FilesWritten = %v, want [README.md Makefile]", out.FilesWritten)
	}
	if len(out.FilesModified) != 1 || out.FilesModified[0] != ".gitignore" {
		t.Errorf("FilesModified = %v, want [.gitignore]", out.FilesModified)
	}
}
