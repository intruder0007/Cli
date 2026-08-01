// Package diag is a minimal, dependency-free logging seam shared by
// core/engine and core/plugin, so those packages can emit diagnostic
// output (what plugin is being spawned, timing, wire-level detail)
// without hardcoding any presentation — cli decides whether/how that
// output is shown (e.g. only with --verbose). core stays free of any
// opinion about terminals, color, or formatting.
package diag

import (
	"fmt"
	"io"
	"time"
)

// Logger receives diagnostic lines. Callers should treat it as
// best-effort, non-critical output — never gate correctness on whether
// a log line was written.
type Logger interface {
	Logf(format string, args ...interface{})
}

// NoopLogger discards everything. The default when no logging is
// requested.
type NoopLogger struct{}

func (NoopLogger) Logf(string, ...interface{}) {}

// WriterLogger writes timestamped lines to w (typically os.Stderr).
type WriterLogger struct {
	W io.Writer
}

func (l WriterLogger) Logf(format string, args ...interface{}) {
	fmt.Fprintf(l.W, "[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
