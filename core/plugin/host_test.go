package plugin

import (
	"bufio"
	"errors"
	"io"
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

func TestSessionCallRPCError(t *testing.T) {
	s := sessionWithResponse(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"boom"}}`)
	err := s.call(time.Second, "plugin.generate", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("got err=%v, want an error mentioning the plugin's message", err)
	}
}
