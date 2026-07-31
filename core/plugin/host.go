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

// Host runs plugins as subprocesses on behalf of the engine.
type Host struct {
	// ShutdownTimeout bounds how long Generate/Apply wait for the plugin
	// process to exit after plugin.shutdown before it's considered hung.
	ShutdownTimeout time.Duration
}

// NewHost returns a Host with sane defaults.
func NewHost() *Host {
	return &Host{ShutdownTimeout: 5 * time.Second}
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

// session wraps one running plugin process for the duration of a single
// generate-or-apply call.
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
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plugin: starting %s: %w", entrypointPath, err)
	}
	return &session{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdinPipe),
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

func (s *session) call(method string, params interface{}, result interface{}) error {
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

	line, err := s.stdout.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("plugin: reading response to %s: %w", method, err)
	}
	var resp rpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
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
	_ = s.call("plugin.shutdown", struct{}{}, nil)
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(h.ShutdownTimeout):
		_ = s.cmd.Process.Kill()
	}
}

// Generate spawns the template plugin at entrypointPath and calls
// plugin.generate.
func (h *Host) Generate(entrypointPath string, req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
	var out sdk.GenerateResponse
	s, err := h.start(entrypointPath)
	if err != nil {
		return out, err
	}
	defer h.finish(s)

	if err := s.call("plugin.initialize", map[string]string{"protocolVersion": sdk.ProtocolVersion}, nil); err != nil {
		return out, err
	}
	if err := s.call("plugin.generate", req, &out); err != nil {
		return out, err
	}
	return out, nil
}

// Apply spawns the capability plugin at entrypointPath and calls
// plugin.apply.
func (h *Host) Apply(entrypointPath string, req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	var out sdk.ApplyResponse
	s, err := h.start(entrypointPath)
	if err != nil {
		return out, err
	}
	defer h.finish(s)

	if err := s.call("plugin.initialize", map[string]string{"protocolVersion": sdk.ProtocolVersion}, nil); err != nil {
		return out, err
	}
	if err := s.call("plugin.apply", req, &out); err != nil {
		return out, err
	}
	return out, nil
}
