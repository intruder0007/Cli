package plugin

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sessionWithResponse builds a session whose stdout is a single canned
// JSON-RPC response line, and whose stdin discards writes — enough to
// test call()/initialize()'s response handling without a real
// subprocess. cmd stays nil, which finish() already treats as "no real
// process to wait on."
func sessionWithResponse(response string) *session {
	return &session{
		stdin:  bufio.NewWriter(io.Discard),
		stdout: bufio.NewReader(strings.NewReader(response + "\n")),
	}
}

func TestHostInitializeSuccess(t *testing.T) {
	h := NewHost()
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"result":{"ok":true,"manifest":{"protocolVersion":"1","name":"plug"}}}`)
	if err := h.initialize(s, "plug", "1"); err != nil {
		t.Errorf("matching protocol+name should succeed, got: %v", err)
	}
}

func TestHostInitializeProtocolMismatch(t *testing.T) {
	h := NewHost()
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"result":{"ok":true,"manifest":{"protocolVersion":"999","name":"plug"}}}`)
	err := h.initialize(s, "plug", "1")
	var mismatch *ProtocolMismatchError
	if !errors.As(err, &mismatch) {
		t.Errorf("got err=%v (%T), want *ProtocolMismatchError", err, err)
	}
}

func TestHostInitializeIdentityMismatch(t *testing.T) {
	h := NewHost()
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"result":{"ok":true,"manifest":{"protocolVersion":"1","name":"wrong-name"}}}`)
	err := h.initialize(s, "expected-name", "1")
	var mismatch *IdentityMismatchError
	if !errors.As(err, &mismatch) {
		t.Errorf("got err=%v (%T), want *IdentityMismatchError", err, err)
	}
}

func TestSessionCallTimeout(t *testing.T) {
	r, _ := io.Pipe() // never written to: the read blocks until timeout
	s := &session{
		stdin:  bufio.NewWriter(io.Discard),
		stdout: bufio.NewReader(r),
	}
	err := s.call(50*time.Millisecond, "plugin.initialize", nil, nil)
	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Errorf("got err=%v (%T), want *TimeoutError", err, err)
	}
}

func TestHostInitializeReportsOKFalse(t *testing.T) {
	h := NewHost()
	// A plugin that answers the handshake but reports ok:false (it
	// validated the request and refuses) must be a hard error, not be
	// treated as success.
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"result":{"ok":false,"manifest":{"protocolVersion":"1","name":"plug"}}}`)
	err := h.initialize(s, "plug", "1")
	if err == nil || !strings.Contains(err.Error(), "initialize") {
		t.Errorf("got err=%v, want an initialize failure error", err)
	}
}

func TestSessionCallRejectsStaleResponseID(t *testing.T) {
	// The request is the session's first, so it gets id 1. A response
	// carrying id 2 is a stale line from an earlier timed-out call and
	// must never be attributed to this request.
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":2,"result":{"ok":true}}`)
	err := s.call(time.Second, "plugin.initialize", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "id 2") || !strings.Contains(err.Error(), "want 1") {
		t.Errorf("got err=%v, want an id mismatch error (got 2, want 1)", err)
	}
}

func TestSessionCallMatchesResponseID(t *testing.T) {
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)
	if err := s.call(time.Second, "plugin.initialize", nil, nil); err != nil {
		t.Errorf("matching response id should succeed, got: %v", err)
	}
}

func TestSessionCallRPCError(t *testing.T) {
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"boom"}}`)
	err := s.call(time.Second, "plugin.generate", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("got err=%v, want an error mentioning the plugin's message", err)
	}
}

// helperCommand builds a command that re-runs this test binary with
// -test.run restricted to TestHelperProcess, so the subprocess runs only
// the scripted helper behavior, not the whole suite.
func helperCommand(t *testing.T, behavior string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"GO_HELPER_BEHAVIOR="+behavior,
	)
	return cmd
}

func helperSession(t *testing.T, behavior string) *session {
	t.Helper()
	cmd := helperCommand(t, behavior)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper process: %v", err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return &session{
		cmd:         cmd,
		stdin:       bufio.NewWriter(stdinPipe),
		stdout:      bufio.NewReader(stdoutPipe),
		stdinCloser: stdinPipe,
	}
}

func TestFinishKillsHungProcess(t *testing.T) {
	h := NewHost()
	h.ShutdownTimeout = 100 * time.Millisecond
	s := helperSession(t, "hang")

	start := time.Now()
	h.finish(s)
	elapsed := time.Since(start)

	// The hang helper ignores plugin.shutdown and never exits on stdin
	// EOF — finish must not block forever on it: it should give up after
	// ShutdownTimeout (shutdown call + wait) and kill the process.
	if elapsed > 5*time.Second {
		t.Fatalf("finish took %s, want it to give up on a hung process", elapsed)
	}
	if elapsed < h.ShutdownTimeout {
		t.Errorf("finish returned after %s, want it to have waited at least ShutdownTimeout (%s) before giving up", elapsed, h.ShutdownTimeout)
	}
	if s.cmd.ProcessState == nil || !s.cmd.ProcessState.Exited() {
		t.Error("hung process still running after finish; it must have been killed")
	}
}

func TestFinishLetsCooperativeProcessExit(t *testing.T) {
	h := NewHost()
	h.ShutdownTimeout = 5 * time.Second
	s := helperSession(t, "exit-on-shutdown")

	start := time.Now()
	h.finish(s)
	elapsed := time.Since(start)

	// The helper answers plugin.shutdown and then exits on stdin EOF (the
	// protocol's documented clean-exit path) — finish must notice the
	// exit well within the timeout instead of killing it.
	if elapsed > 2*time.Second {
		t.Errorf("finish took %s for a process that answers shutdown and exits on stdin EOF", elapsed)
	}
	if s.cmd.ProcessState == nil || !s.cmd.ProcessState.Exited() {
		t.Error("cooperative process still running after finish")
	}
}

func TestHostStartWiresStderrToPluginProcess(t *testing.T) {
	// Build a minimal entrypoint that writes one line to stderr and
	// exits, then verify Host.start() wires Host.Stderr through to the
	// subprocess (protocol-reserved for plugin logs).
	dir := t.TempDir()
	src := filepath.Join(dir, "stderr_helper.go")
	if err := os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "hello from plugin stderr")
	os.Exit(0)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, exeName("stderr-plugin"))
	if out, err := exec.Command("go", "build", "-o", exe, src).CombinedOutput(); err != nil {
		t.Fatalf("building stderr helper: %v\n%s", err, out)
	}

	h := NewHost()
	var stderr bytes.Buffer
	h.Stderr = &stderr

	s, err := h.start(exe)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	h.finish(s) // helper exits on its own; finish's wait path reaps it

	if !strings.Contains(stderr.String(), "hello from plugin stderr") {
		t.Errorf("plugin stderr = %q, want it wired through Host.Stderr", stderr.String())
	}
	if s.cmd.ProcessState == nil || !s.cmd.ProcessState.Exited() {
		t.Error("helper process still running after finish")
	}
}

// TestHelperProcess is not a real test: it's the scripted behavior for
// the helper-command tests above. It's only run when invoked as a
// subprocess (GO_WANT_HELPER_PROCESS=1) with -test.run restricted to it.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	switch os.Getenv("GO_HELPER_BEHAVIOR") {
	case "hang":
		_, _ = io.Copy(io.Discard, os.Stdin)
		select {} // ignore the clean-exit signal too
	case "exit-on-shutdown":
		// Answer the shutdown call (a fresh session's first call gets
		// id 1), then block on stdin until the host closes it and exit.
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if line != "" {
			fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":1,"result":{}}`)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper behavior %q\n", os.Getenv("GO_HELPER_BEHAVIOR"))
		os.Exit(1)
	}
}
