// Command bootstrap is the Cli interactive/non-interactive wizard. See
// docs/cli/usage.md and ADR-0007.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/intruder0007/Cli/cli/internal/prompt"
	"github.com/intruder0007/Cli/core/config"
	"github.com/intruder0007/Cli/core/diag"
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
		fmt.Fprintln(os.Stderr, prompt.HelpText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "plugins":
		cmdPlugins(os.Args[2:])
	case "config":
		cmdConfig(os.Args[2:])
	case "doctor":
		cmdDoctor(os.Args[2:])
	case "version":
		fmt.Printf("bootstrap version %s (%s, %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	case "-h", "--help", "help":
		fmt.Println(prompt.HelpText)
	default:
		fmt.Fprintln(os.Stderr, prompt.HelpText)
		os.Exit(1)
	}
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
	projectName, rest := extractProjectName(args, map[string]bool{
		"-no-color": true, "--no-color": true,
		"-verbose": true, "--verbose": true,
		"-v": true, "--v": true,
	})

	fs := flag.NewFlagSet("new", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: bootstrap new [project-name] [flags]

Generates a new project. With no flags and no --answers, runs the
interactive wizard. Flags below make it non-interactive.`)
		fs.PrintDefaults()
	}
	theme := fs.String("theme", "", "CLI theme: default or minimal")
	projectType := fs.String("project-type", "", "project type, e.g. backend-service")
	language := fs.String("language", "", "language, e.g. go")
	framework := fs.String("framework", "", "framework, e.g. rest-api")
	caps := fs.String("capabilities", "", "comma-separated capability ids")
	answersFile := fs.String("answers", "", "path to an answers file")
	noColor := fs.Bool("no-color", false, "disable color output")
	verbose := fs.Bool("verbose", false, "print diagnostic logging (plugin spawn/timing) to stderr")
	fs.BoolVar(verbose, "v", false, "shorthand for -verbose")
	fs.Parse(rest)

	noColorEnv := os.Getenv("NO_COLOR") != ""
	interactive := *answersFile == "" && projectName == "" && *theme == "" && *projectType == "" && *language == "" && *framework == "" && *caps == ""

	var a config.Answers
	var err error

	switch {
	case *answersFile != "":
		a, err = prompt.ParseAnswersFile(*answersFile)
	case !interactive:
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
		// Banner uses the persisted theme if one exists (a wizard run
		// always saves its own choice — see wizard.go), else "default".
		// The wizard's own theme question can still change it.
		bannerTheme := "default"
		if cfg, cfgErr := prompt.LoadConfig(); cfgErr == nil && cfg.Theme != "" {
			bannerTheme = cfg.Theme
		}
		prompt.Banner(os.Stdout, prompt.GetTheme(bannerTheme, *noColor || noColorEnv))
		a, err = prompt.RunWizard(os.Stdout)
	}

	t := prompt.GetTheme(a.Theme, *noColor || noColorEnv)

	if err != nil {
		if err == prompt.ErrCancelled {
			fmt.Fprintln(os.Stdout, t.Info("cancelled"))
			os.Exit(130)
		}
		prompt.ErrorScreen(os.Stdout, t, err)
		os.Exit(1)
	}

	targetDir, err := filepath.Abs(a.ProjectName)
	if err != nil {
		prompt.ErrorScreen(os.Stdout, t, err)
		os.Exit(1)
	}

	var logger diag.Logger = diag.NoopLogger{}
	if *verbose {
		logger = diag.WriterLogger{W: os.Stderr}
	}

	reg := registry.New(pluginDirs()...)
	host := plugin.NewHost()
	host.Logger = logger
	eng := engine.New(reg, host)
	eng.Logger = logger

	summary, err := eng.Run(targetDir, a)
	if err != nil {
		prompt.ErrorScreen(os.Stdout, t, err)
		os.Exit(1)
	}

	prompt.SuccessScreen(os.Stdout, t, a.ProjectName, summary)
}

func cmdPlugins(args []string) {
	fs := flag.NewFlagSet("plugins", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bootstrap plugins list")
	}
	fs.Parse(args)

	if fs.NArg() == 0 || fs.Arg(0) != "list" {
		fs.Usage()
		os.Exit(1)
	}

	reg := registry.New(pluginDirs()...)
	found, issues, err := reg.DiscoverWithIssues()
	if err != nil {
		prompt.ErrorScreen(os.Stdout, prompt.GetTheme("default", os.Getenv("NO_COLOR") != ""), err)
		os.Exit(1)
	}
	for _, issue := range issues {
		fmt.Fprintf(os.Stderr, "skipped %s: %v\n", issue.Path, issue.Err)
	}
	if len(found) == 0 {
		fmt.Println("no plugins found")
		return
	}
	for _, p := range found {
		fmt.Printf("%s\t%s\t%s\t%s\n", p.Manifest.Name, p.Manifest.Kind, p.Manifest.Version, p.Manifest.DisplayName)
	}
}

// cmdDoctor runs a small set of local health checks — plugin directory
// resolution and manifest validity — and reports a pass/fail summary
// with actionable hints. It never spawns any plugin subprocess (that
// happens only during `new`); it only exercises discovery, the same
// fail-fast surface `plugins list` uses.
func cmdDoctor(args []string) {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: bootstrap doctor") }
	fs.Parse(args)

	t := prompt.GetTheme("default", os.Getenv("NO_COLOR") != "")
	ok := true

	dirs := pluginDirs()
	fmt.Println(t.Header("Plugin directories:"))
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			fmt.Println(t.Success("  " + abs))
		} else {
			fmt.Println(t.Dim("  " + abs + " (not found, skipped)"))
		}
	}

	reg := registry.New(dirs...)
	found, issues, err := reg.DiscoverWithIssues()
	if err != nil {
		fmt.Println(t.Failure("discovery failed: " + err.Error()))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(t.Header("Plugins:"))
	if len(found) == 0 {
		fmt.Println(t.Dim("  none discovered"))
		ok = false
	}
	for _, p := range found {
		fmt.Println(t.Success(fmt.Sprintf("  %s (%s) v%s", p.Manifest.Name, p.Manifest.Kind, p.Manifest.Version)))
	}
	for _, issue := range issues {
		fmt.Println(t.Failure(fmt.Sprintf("  %s: %v", issue.Path, issue.Err)))
		ok = false
	}

	fmt.Println()
	if ok {
		fmt.Println(t.Success("doctor: all checks passed"))
		return
	}
	fmt.Println(t.Failure("doctor: found issues"))
	fmt.Println(t.Dim("  hint: check plugin.json files against docs/plugins/authoring.md / docs/templates/authoring.md, or set CLI_PLUGIN_DIRS to point at the right directories."))
	os.Exit(1)
}

func configUsage() {
	fmt.Fprintln(os.Stderr, `usage: bootstrap config get theme
       bootstrap config set theme <default|minimal>`)
}

func cmdConfig(args []string) {
	if len(args) < 2 || (args[0] != "get" && args[0] != "set") || args[1] != "theme" {
		configUsage()
		os.Exit(1)
	}
	if args[0] == "get" {
		cmdConfigGetTheme()
	} else {
		cmdConfigSetTheme(args[2:])
	}
}

func cmdConfigGetTheme() {
	cfg, err := prompt.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(cfg.Theme)
}

func cmdConfigSetTheme(rest []string) {
	if len(rest) < 1 {
		configUsage()
		os.Exit(1)
	}
	name := rest[0]
	if !isValidThemeName(name) {
		fmt.Fprintf(os.Stderr, "unknown theme %q (want one of: %s)\n", name, strings.Join(prompt.ThemeNames(), ", "))
		os.Exit(1)
	}
	cfg, _ := prompt.LoadConfig()
	cfg.Theme = name
	if err := prompt.SaveConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func isValidThemeName(name string) bool {
	for _, n := range prompt.ThemeNames() {
		if n == name {
			return true
		}
	}
	return false
}
