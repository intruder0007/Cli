// Package sdk implements the plugin side of the Cli plugin protocol:
// subprocess + line-delimited JSON-RPC 2.0 over stdio. See
// docs/architecture/plugin-protocol.md for the full wire spec.
//
// Stability: this package is a Stable public API (see
// docs/architecture/api-compatibility.md and ADR-0013). Every exported
// identifier — types, fields, function signatures, and constants — is a
// contract. Breaking changes require the deprecation period and
// versioning rules described there; additive changes (new functions,
// new optional fields) are ordinary PRs.
package sdk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Manifest mirrors a plugin's plugin.json.
type Manifest struct {
	ProtocolVersion string `json:"protocolVersion"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Kind            string `json:"kind"` // "template" | "capability"
	DisplayName     string `json:"displayName"`
	ProjectType     string `json:"projectType,omitempty"`
	Language        string `json:"language,omitempty"`
	Framework       string `json:"framework,omitempty"`
	CapabilityID    string `json:"capabilityId,omitempty"`
	// DependsOn names other capabilityIds that must be applied before
	// this one, among whichever capabilities the user actually selected
	// (an entry naming a capability the user didn't select is ignored —
	// V1 never auto-installs capabilities). Capabilities only.
	DependsOn []string `json:"dependsOn,omitempty"`
	// SupportedPlatforms restricts a template to specific GOOS values
	// ("linux", "darwin", "windows"). Empty means all platforms.
	// Templates only.
	SupportedPlatforms []string `json:"supportedPlatforms,omitempty"`
	Entrypoint         string   `json:"entrypoint"`
}

// Validate checks that a manifest has the fields required for its kind.
// Used both by core/registry (discovering third-party manifests on disk)
// and by Serve (a plugin validating its own loaded manifest).
func (m Manifest) Validate() error {
	if m.ProtocolVersion == "" {
		return fmt.Errorf("manifest: protocolVersion is required")
	}
	if m.Name == "" {
		return fmt.Errorf("manifest: name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest: version is required")
	}
	if m.Entrypoint == "" {
		return fmt.Errorf("manifest: entrypoint is required")
	}
	switch m.Kind {
	case "template":
		if m.ProjectType == "" || m.Language == "" || m.Framework == "" {
			return fmt.Errorf("manifest: template %q requires projectType, language, and framework", m.Name)
		}
	case "capability":
		if m.CapabilityID == "" {
			return fmt.Errorf("manifest: capability %q requires capabilityId", m.Name)
		}
	default:
		return fmt.Errorf("manifest: %q has unknown kind %q (want \"template\" or \"capability\")", m.Name, m.Kind)
	}
	return nil
}

// GenerateRequest is sent to plugin.generate (templates only).
type GenerateRequest struct {
	TargetDir   string            `json:"targetDir"`
	ProjectName string            `json:"projectName"`
	Answers     map[string]string `json:"answers"`
}

// GenerateResponse is the result of plugin.generate.
type GenerateResponse struct {
	FilesWritten []string `json:"filesWritten"`
	NextSteps    []string `json:"nextSteps,omitempty"`
}

// ApplyRequest is sent to plugin.apply (capabilities only).
type ApplyRequest struct {
	TargetDir   string            `json:"targetDir"`
	ProjectName string            `json:"projectName"`
	Answers     map[string]string `json:"answers"`
}

// ApplyResponse is the result of plugin.apply.
type ApplyResponse struct {
	FilesWritten  []string `json:"filesWritten"`
	FilesModified []string `json:"filesModified,omitempty"`
	NextSteps     []string `json:"nextSteps,omitempty"`
}

// TemplatePlugin is implemented by plugins with kind "template".
type TemplatePlugin interface {
	Generate(GenerateRequest) (GenerateResponse, error)
}

// CapabilityPlugin is implemented by plugins with kind "capability".
type CapabilityPlugin interface {
	Apply(ApplyRequest) (ApplyResponse, error)
}

const ProtocolVersion = "1"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

// Serve runs the plugin's JSON-RPC/stdio loop until plugin.shutdown or EOF.
// plugin must implement TemplatePlugin, CapabilityPlugin, or both.
func Serve(plugin interface{}) {
	manifest, err := loadManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdk: loading plugin.json: %v\n", err)
		os.Exit(1)
	}
	if err := manifest.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "sdk: invalid plugin.json: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "sdk: reading request: %v\n", err)
			return
		}

		var req rpcRequest
		if unmarshalErr := json.Unmarshal(line, &req); unmarshalErr != nil {
			writeResponse(writer, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: unmarshalErr.Error()}})
			continue
		}

		resp := dispatch(plugin, manifest, req)
		writeResponse(writer, resp)

		if req.Method == "plugin.shutdown" {
			return
		}
		if err == io.EOF {
			return
		}
	}
}

func dispatch(plugin interface{}, manifest Manifest, req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "plugin.initialize":
		resp.Result = map[string]interface{}{"ok": true, "manifest": manifest}

	case "plugin.generate":
		tp, ok := plugin.(TemplatePlugin)
		if !ok {
			resp.Error = &rpcError{Code: -32601, Message: "plugin does not implement generate"}
			return resp
		}
		var genReq GenerateRequest
		if err := json.Unmarshal(req.Params, &genReq); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: err.Error()}
			return resp
		}
		out, err := tp.Generate(genReq)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp
		}
		resp.Result = out

	case "plugin.apply":
		cp, ok := plugin.(CapabilityPlugin)
		if !ok {
			resp.Error = &rpcError{Code: -32601, Message: "plugin does not implement apply"}
			return resp
		}
		var applyReq ApplyRequest
		if err := json.Unmarshal(req.Params, &applyReq); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: err.Error()}
			return resp
		}
		out, err := cp.Apply(applyReq)
		if err != nil {
			resp.Error = &rpcError{Code: -32000, Message: err.Error()}
			return resp
		}
		resp.Result = out

	case "plugin.shutdown":
		resp.Result = map[string]interface{}{"ok": true}

	default:
		resp.Error = &rpcError{Code: -32601, Message: "unknown method: " + req.Method}
	}

	return resp
}

func writeResponse(w io.Writer, resp rpcResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	w.Write(b)
	w.Write([]byte("\n"))
}

// loadManifest reads plugin.json from the same directory as the running
// executable.
func loadManifest() (Manifest, error) {
	var m Manifest
	exe, err := os.Executable()
	if err != nil {
		return m, err
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "plugin.json"))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}
