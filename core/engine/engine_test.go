package engine

import (
	"errors"
	"testing"

	"github.com/intruder0007/Cli/core/config"
	"github.com/intruder0007/Cli/core/registry"
	sdk "github.com/intruder0007/Cli/sdk/go/sdk"
)

func capPlugin(id string, dependsOn ...string) registry.Plugin {
	return registry.Plugin{
		Manifest: sdk.Manifest{
			Name: id, Kind: "capability", CapabilityID: id, DependsOn: dependsOn,
		},
	}
}

func TestSortByDependenciesNoConstraintsPreservesOrder(t *testing.T) {
	caps := []registry.Plugin{capPlugin("a"), capPlugin("b"), capPlugin("c")}
	got, err := sortByDependencies(caps)
	if err != nil {
		t.Fatalf("sortByDependencies: %v", err)
	}
	want := []string{"a", "b", "c"}
	for i, w := range want {
		if got[i].Manifest.CapabilityID != w {
			t.Fatalf("got order %v, want %v (no dependsOn declared -> user's selection order preserved)", idsOf(got), want)
		}
	}
}

func TestSortByDependenciesRespectsEdges(t *testing.T) {
	// User selected in order a, b, c, but b depends on c.
	caps := []registry.Plugin{capPlugin("a"), capPlugin("b", "c"), capPlugin("c")}
	got, err := sortByDependencies(caps)
	if err != nil {
		t.Fatalf("sortByDependencies: %v", err)
	}
	pos := map[string]int{}
	for i, p := range got {
		pos[p.Manifest.CapabilityID] = i
	}
	if pos["c"] >= pos["b"] {
		t.Errorf("got order %v, want c before b (b depends on c)", idsOf(got))
	}
}

func TestSortByDependenciesIgnoresUnselectedDependency(t *testing.T) {
	// b depends on "z", which the user never selected — must be ignored,
	// not treated as an error or missing dependency.
	caps := []registry.Plugin{capPlugin("a"), capPlugin("b", "z")}
	got, err := sortByDependencies(caps)
	if err != nil {
		t.Fatalf("sortByDependencies with an unselected dependency should not error, got: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d capabilities, want 2", len(got))
	}
}

func TestSortByDependenciesDetectsCycle(t *testing.T) {
	caps := []registry.Plugin{capPlugin("a", "b"), capPlugin("b", "a")}
	_, err := sortByDependencies(caps)
	var cycleErr *CapabilityCycleError
	if !errors.As(err, &cycleErr) {
		t.Errorf("got err=%v (%T), want *CapabilityCycleError", err, err)
	}
}

func idsOf(caps []registry.Plugin) []string {
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = c.Manifest.CapabilityID
	}
	return out
}

// fakeResolver and fakeRunner let engine.Run be tested without any real
// plugin discovery or subprocess.
type fakeResolver struct {
	template     registry.Plugin
	templateErr  error
	capabilities map[string]registry.Plugin
	capErrs      map[string]error
	capCalls     []string // capability ids passed to ResolveCapability, in order
}

func (f *fakeResolver) ResolveTemplate(projectType, language, framework string) (registry.Plugin, error) {
	return f.template, f.templateErr
}

func (f *fakeResolver) ResolveCapability(id string) (registry.Plugin, error) {
	f.capCalls = append(f.capCalls, id)
	if err, ok := f.capErrs[id]; ok {
		return registry.Plugin{}, err
	}
	return f.capabilities[id], nil
}

type fakeRunner struct {
	generateCalls int
	applyCalls    []string // capability names, in call order
}

func (f *fakeRunner) Generate(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.GenerateRequest) (sdk.GenerateResponse, error) {
	f.generateCalls++
	return sdk.GenerateResponse{FilesWritten: []string{"go.mod"}}, nil
}

func (f *fakeRunner) Apply(entrypointPath, expectedName, expectedProtocolVersion string, req sdk.ApplyRequest) (sdk.ApplyResponse, error) {
	f.applyCalls = append(f.applyCalls, expectedName)
	return sdk.ApplyResponse{}, nil
}

func validAnswers(caps ...string) config.Answers {
	return config.Answers{
		ProjectName: "demo", Theme: "default",
		ProjectType: "backend-service", Language: "go", Framework: "rest-api",
		Capabilities: caps,
	}
}

func TestRunFailsFastBeforeAnySideEffect(t *testing.T) {
	resolver := &fakeResolver{
		template: registry.Plugin{Manifest: sdk.Manifest{Name: "tmpl"}},
		capabilities: map[string]registry.Plugin{
			"good": {Manifest: sdk.Manifest{Name: "good", CapabilityID: "good"}},
		},
		capErrs: map[string]error{"bad": errors.New("no such capability")},
	}
	runner := &fakeRunner{}
	eng := New(resolver, runner)

	// "good" is selected before "bad" — if resolution weren't fail-fast,
	// the template (and "good") would already have run by the time "bad"
	// fails to resolve.
	_, err := eng.Run(t.TempDir(), validAnswers("good", "bad"))
	if err == nil {
		t.Fatal("Run with an unresolvable capability should fail, got nil error")
	}
	if runner.generateCalls != 0 {
		t.Errorf("Generate was called %d times, want 0 — resolution must happen before any execution", runner.generateCalls)
	}
	if len(runner.applyCalls) != 0 {
		t.Errorf("Apply was called for %v, want none — resolution must happen before any execution", runner.applyCalls)
	}
}

func TestRunCollapsesDuplicateCapabilitySelections(t *testing.T) {
	resolver := &fakeResolver{
		template: registry.Plugin{Manifest: sdk.Manifest{Name: "tmpl"}},
		capabilities: map[string]registry.Plugin{
			"a": {Manifest: sdk.Manifest{Name: "a", CapabilityID: "a"}},
			"b": {Manifest: sdk.Manifest{Name: "b", CapabilityID: "b"}},
		},
	}
	runner := &fakeRunner{}
	eng := New(resolver, runner)

	// Duplicate ids in the selection must be collapsed (first occurrence
	// wins) — both in resolution and in apply order — so the capability
	// isn't run twice against the same target directory.
	if _, err := eng.Run(t.TempDir(), validAnswers("a", "b", "a", "b")); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := len(runner.applyCalls); got != 2 {
		t.Errorf("Apply called %d times, want 2 (a and b once each), calls = %v", got, runner.applyCalls)
	}
	if got := len(resolver.capCalls); got != 2 {
		t.Errorf("ResolveCapability called %d times, want 2 (one per unique id), ids = %v", got, resolver.capCalls)
	}
}

func TestRunAppliesCapabilitiesInDependencyOrder(t *testing.T) {
	resolver := &fakeResolver{
		template: registry.Plugin{Manifest: sdk.Manifest{Name: "tmpl"}},
		capabilities: map[string]registry.Plugin{
			"a": {Manifest: sdk.Manifest{Name: "a", CapabilityID: "a"}},
			"b": {Manifest: sdk.Manifest{Name: "b", CapabilityID: "b", DependsOn: []string{"a"}}},
		},
	}
	runner := &fakeRunner{}
	eng := New(resolver, runner)

	// Select "b" before "a" in the answers — dependency order must still
	// put "a" first.
	if _, err := eng.Run(t.TempDir(), validAnswers("b", "a")); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(runner.applyCalls) != 2 || runner.applyCalls[0] != "a" || runner.applyCalls[1] != "b" {
		t.Errorf("apply order = %v, want [a b] (b depends on a)", runner.applyCalls)
	}
}
