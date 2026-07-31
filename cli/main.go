// Command bootstrap is the Cli interactive/non-interactive wizard. See
// docs/cli/usage.md.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intruder0007/Cli/cli/internal/prompt"
	"github.com/intruder0007/Cli/core/config"
	"github.com/intruder0007/Cli/core/engine"
	"github.com/intruder0007/Cli/core/plugin"
	"github.com/intruder0007/Cli/core/registry"
)

// version is overridden at build time via:
//
//	go build -ldflags "-X main.version=vX.Y.Z" ./cli
//
// See ADR-0006 and .github/workflows/release.yml. Left as "dev" for
// ordinary local builds (go run, go build with no ldflags, go install).
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "plugins":
		cmdPlugins(os.Args[2:])
	case "version":
		fmt.Println("bootstrap version " + version)
	case "-h", "--help", "help":
		printUsage()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage: bootstrap <command> [flags]

commands:
  new [project-name]   generate a new project (interactive if no flags/answers given)
  plugins list          list discovered template and capability plugins
  version                print the CLI version`)
}

// pluginDirs returns the local directories the registry scans: an
// explicit CLI_PLUGIN_DIRS override (os.PathListSeparator-delimited, for
// installed binaries or tests where neither the executable's directory
// nor the working directory is the repo root) takes priority, then
// directories relative to the running executable, then the current
// working directory (matching a repo-root `./bin/bootstrap new` dev
// workflow). Deduplicated by absolute path, since the executable's
// directory and the working directory are the same thing for the most
// common real usage — cd into an extracted release archive and run
// ./bootstrap — which would otherwise register every plugin twice.
func pluginDirs() []string {
	if override := os.Getenv("CLI_PLUGIN_DIRS"); override != "" {
		return dedupeAbs(strings.Split(override, string(os.PathListSeparator)))
	}

	dirs := []string{filepath.Join("templates"), filepath.Join("plugins", "builtin")}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append([]string{
			filepath.Join(exeDir, "templates"),
			filepath.Join(exeDir, "plugins", "builtin"),
		}, dirs...)
	}
	return dedupeAbs(dirs)
}

func dedupeAbs(dirs []string) []string {
	seen := make(map[string]bool, len(dirs))
	var out []string
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, d)
	}
	return out
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok != "" {
			out = append(out, tok)
		}
	}
	return out
}

// extractProjectName pulls the positional project-name argument out of
// args regardless of where it appears relative to flags (docs/cli/usage.md
// documents "bootstrap new my-project --theme ..."), since the stdlib
// flag package stops parsing at the first non-flag token and would
// otherwise silently swallow every flag after a leading positional arg.
func extractProjectName(args []string, boolFlags map[string]bool) (string, []string) {
	var rest []string
	projectName := ""
	skipNext := false
	for _, a := range args {
		if skipNext {
			rest = append(rest, a)
			skipNext = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			rest = append(rest, a)
			if !boolFlags[a] && !strings.Contains(a, "=") {
				skipNext = true
			}
			continue
		}
		if projectName == "" {
			projectName = a
			continue
		}
		rest = append(rest, a)
	}
	return projectName, rest
}

func cmdNew(args []string) {
	projectName, rest := extractProjectName(args, map[string]bool{"-no-color": true, "--no-color": true})

	fs := flag.NewFlagSet("new", flag.ExitOnError)
	theme := fs.String("theme", "", "CLI theme: default or minimal")
	projectType := fs.String("project-type", "", "project type, e.g. backend-service")
	language := fs.String("language", "", "language, e.g. go")
	framework := fs.String("framework", "", "framework, e.g. rest-api")
	caps := fs.String("capabilities", "", "comma-separated capability ids")
	answersFile := fs.String("answers", "", "path to an answers file")
	noColor := fs.Bool("no-color", false, "disable color output")
	fs.Parse(rest)

	noColorEnv := os.Getenv("NO_COLOR") != ""

	var a config.Answers
	var err error

	switch {
	case *answersFile != "":
		a, err = prompt.ParseAnswersFile(*answersFile)
	case projectName != "" || *theme != "" || *projectType != "" || *language != "" || *framework != "" || *caps != "":
		if *theme == "" {
			*theme = "default"
		}
		a = config.Answers{
			ProjectName:  projectName,
			Theme:        *theme,
			ProjectType:  *projectType,
			Language:     *language,
			Framework:    *framework,
			Capabilities: splitCSV(*caps),
		}
	default:
		a, err = prompt.RunWizard(bufio.NewReader(os.Stdin), os.Stdout)
	}

	renderer := prompt.NewRenderer(a.Theme, *noColor || noColorEnv)

	if err != nil {
		fmt.Fprintln(os.Stderr, renderer.Failure(err.Error()))
		os.Exit(1)
	}

	targetDir, err := filepath.Abs(a.ProjectName)
	if err != nil {
		fmt.Fprintln(os.Stderr, renderer.Failure(err.Error()))
		os.Exit(1)
	}

	reg := registry.New(pluginDirs()...)
	eng := engine.New(reg, plugin.NewHost())

	summary, err := eng.Run(targetDir, a)
	if err != nil {
		fmt.Fprintln(os.Stderr, renderer.Failure(err.Error()))
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, renderer.Success(fmt.Sprintf("Generated %s", a.ProjectName)))
	for _, f := range summary.FilesWritten {
		fmt.Fprintln(os.Stdout, renderer.Info("  + "+f))
	}
	if len(summary.NextSteps) > 0 {
		fmt.Fprintln(os.Stdout, renderer.Header("Next steps:"))
		for _, s := range summary.NextSteps {
			fmt.Fprintln(os.Stdout, "  "+s)
		}
	}
}

func cmdPlugins(args []string) {
	fs := flag.NewFlagSet("plugins", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() == 0 || fs.Arg(0) != "list" {
		fmt.Fprintln(os.Stderr, "usage: bootstrap plugins list")
		os.Exit(1)
	}

	reg := registry.New(pluginDirs()...)
	found, err := reg.Discover()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(found) == 0 {
		fmt.Println("no plugins found")
		return
	}
	for _, p := range found {
		fmt.Printf("%s\t%s\t%s\t%s\n", p.Manifest.Name, p.Manifest.Kind, p.Manifest.Version, p.Manifest.DisplayName)
	}
}
