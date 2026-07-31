// Package plugin is the client side of the plugin protocol: it spawns a
// plugin's entrypoint as a subprocess and speaks line-delimited JSON-RPC
// 2.0 over its stdio. See docs/architecture/plugin-protocol.md.
package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
)

const defaultCallTimeout = 30 * time.Second

// Host runs plugins as subprocesses on behalf of the engine.
type Host struct {
	// ShutdownTimeout bounds how long Generate/Apply wait for the plugin
	// process to exit after plugin.shutdown before it's considered hung.
	ShutdownTimeout time.Duration
	// CallTimeout bounds how long a single JSON-RPC call (initialize,
	// generate, apply) waits for a response before the plugin is
	// considered hung and killed. Defaults to 30s if zero.
	CallTimeout time.Duration
}

// NewHost returns a Host with sane defaults.
func NewHost() *Host {
	return &Host{ShutdownTimeout: 5 * time.Second, CallTimeout: defaultCallTimeout}
}

func (h *Host) callTimeout() time.Duration {
	if h.CallTimeout > 0 {
		return h.CallTimeout
	}
	return defaultCallTimeout
}

// StartError wraps a failure to spawn a plugin's entrypoint.
type StartError struct {
	EntrypointPath string
	Err            error
}

func (e *StartError) Error() string {
	return fmt.Sprintf("plugin: starting %s: %v", e.EntrypointPath, e.Err)
}
func (e *StartError) Unwrap() error { return e.Err }

// ProtocolMismatchError means the running plugin process reported a
// different protocolVersion than its on-disk manifest declared.
type ProtocolMismatchError struct {
	PluginName, Want, Got string
}

func (e *ProtocolMismatchError) Error() string {
	return fmt.Sprintf("plugin %q protocol version mismatch: host expects %q, plugin reported %q", e.PluginName, e.Want, e.Got)
}

// IdentityMismatchError means the running plugin process reported a
// different name than the manifest the registry discovered it under —
// e.g. a stale or swapped binary sitting where a different plugin's
// entrypoint was expected.
type IdentityMismatchError struct {
	Expected, Got string
}

func (e *IdentityMismatchError) Error() string {
	return fmt.Sprintf("plugin identity mismatch: expected %q (from its manifest on disk), running process reported %q", e.Expected, e.Got)
}

// TimeoutError means a plugin didn't respond to a JSON-RPC call within
// CallTimeout; the process is killed.
type TimeoutError struct {
	Method  string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("plugin: %s timed out after %s", e.Method, e.Timeout)
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type initializeResult struct {
	OK       bool         `json:"ok"`
	Manifest sdk.Manifest `json:"manifest"`
}

// session wraps one running plugin process for the duration of a single
// generate-or-apply call. cmd is nil in tests that construct a session
// directly around an io.Pipe rather than a real subprocess — call()
// itself never touches cmd, only finish() does.
type session struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Reader
	nextID int
}

func (h *Host) start(entrypointPath string) (*session, error) {
	cmd := exec.Command(entrypointPath)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, &StartError{EntrypointPath: entrypointPath, Err: err}
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &StartError{EntrypointPath: entrypointPath, Err: err}
	}
	if err := cmd.Start(); err != nil {
		return nil, &StartError{EntrypointPath: entrypointPath, Err: err}
	}
	return &session{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdinPipe),
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

// call sends a JSON-RPC request and waits for its response, racing the
// (blocking) read against timeout so a hung plugin can't block the host
// forever. The channel is buffered so the reader goroutine never leaks
// even if nobody's listening after a timeout fires.
func (s *session) call(timeout time.Duration, method string, params interface{}, result interface{}) error {
	s.nextID++
	req := rpcRequest{JSONRPC: "2.0", ID: s.nextID, Method: method, Params: params}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := s.stdin.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := s.stdin.Flush(); err != nil {
		return err
	}

	type readOutcome struct {
		line []byte
		err  error
	}
	ch := make(chan readOutcome, 1)
	go func() {
		line, err := s.stdout.ReadBytes('\n')
		ch <- readOutcome{line, err}
	}()

	var outcome readOutcome
	select {
	case outcome = <-ch:
	case <-time.After(timeout):
		return &TimeoutError{Method: method, Timeout: timeout}
	}
	if outcome.err != nil {
		return fmt.Errorf("plugin: reading response to %s: %w", method, outcome.err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(outcome.line, &resp); err != nil {
		return fmt.Errorf("plugin: decoding response to %s: %w", method, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("plugin: %s failed: %s", method, resp.Error.Message)
	}
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("plugin: decoding result of %s: %w", method, err)
		}
	}
	return nil
}

func (h *Host) finish(s *session) {
	if s.cmd == nil {
		return // test session with no real process
	}
	_ = s.call(h.callTimeout(), "plugin.shutdown", struct{}{}, nil)
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(h.ShutdownTimeout):
		_ = s.cmd.Process.Kill()
	}
}

// initialize performs the plugin.initialize handshake and cross-checks
// the running process's self-reported manifest against what the
// registry discovered on disk (expectedName, expectedProtocolVersion) —
// catching a stale or swapped binary.
func (h *Host) initialize(s *session, expectedName, expectedProtocolVersion string) error {
	var result initializeResult
	if err := s.call(h.callTimeout(), "plugin.initialize", map[string]string{"protocolVersion": sdk.ProtocolVersion}, &result); err != nil {
		return err
	}
	if result.Manifest.ProtocolVersion != expectedProtocolVersion {
		return &ProtocolMismatchError{PluginName: expectedName, Want: expectedProtocolVersion, Got: result.Manifest.ProtocolVersion}
	}
	if result.Manifest.Name != expectedName {
		return &IdentityMismatchError{Expected: expectedName, Got: result.Manifest.Name}
	}
	return nil
}

// Generate spawns the template plugin at entrypointPath and calls
// plugin.generate. expectedName/expectedProtocolVersion come from the
// manifest the registry discovered on disk, for the identity/protocol
// cross-check in initialize.
func (h *Host) Generate(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
	var out sdk.GenerateResponse
	s, err := h.start(entrypointPath)
	if err != nil {
		return out, err
	}
	defer h.finish(s)

	if err := h.initialize(s, expectedName, expectedProtocolVersion); err != nil {
		return out, err
	}
	if err := s.call(h.callTimeout(), "plugin.generate", req, &out); err != nil {
		return out, err
	}
	return out, nil
}

// Apply spawns the capability plugin at entrypointPath and calls
// plugin.apply. expectedName/expectedProtocolVersion come from the
// manifest the registry discovered on disk, for the identity/protocol
// cross-check in initialize.
func (h *Host) Apply(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	var out sdk.ApplyResponse
	s, err := h.start(entrypointPath)
	if err != nil {
		return out, err
	}
	defer h.finish(s)

	if err := h.initialize(s, expectedName, expectedProtocolVersion); err != nil {
		return out, err
	}
	if err := s.call(h.callTimeout(), "plugin.apply", req, &out); err != nil {
		return out, err
	}
	return out, nil
}
