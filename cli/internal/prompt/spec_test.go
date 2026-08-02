package prompt

import "testing"

func specFixture() WizardSpec {
	return WizardSpec{
		Templates: []TemplateSpec{
			{"backend-service", "go", "rest-api", "Go REST API Service"},
			{"backend-service", "node", "http-api", "Node.js HTTP API Service"},
			{"backend-service", "go", "grpc", "Go gRPC Service"},
			{"web-app", "node", "next", "Next.js Web App"},
		},
		Capabilities: []CapabilitySpec{
			{"git-init", "Initialize Git repository"},
			{"readme", "Enhance README"},
		},
	}
}

func idsOf(opts []option) []string {
	out := make([]string, len(opts))
	for i, o := range opts {
		out[i] = o.ID
	}
	return out
}

func TestProjectTypeOptionsDedupeAndHumanize(t *testing.T) {
	got := specFixture().projectTypeOptions()
	want := []string{"backend-service", "web-app"}
	for i, w := range want {
		if got[i].ID != w {
			t.Fatalf("project type %d: got %q, want %q (full: %v)", i, got[i].ID, want, idsOf(got))
		}
	}
	if !got[0].Available {
		t.Error("every registry-derived option must be available")
	}
	if got[0].Name != "Backend Service" {
		t.Errorf("humanized name: got %q, want %q", got[0].Name, "Backend Service")
	}
}

func TestLanguageOptionsFilteredByProjectType(t *testing.T) {
	backend := specFixture().languageOptions("backend-service")
	if got, want := idsOf(backend), []string{"go", "node"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("backend-service languages = %v, want %v", got, want)
	}
	if backend[0].Name != "Go" || backend[1].Name != "Node.js" {
		t.Errorf("language display names = %q, %q; want Go, Node.js", backend[0].Name, backend[1].Name)
	}

	web := specFixture().languageOptions("web-app")
	if got, want := idsOf(web), []string{"node"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("web-app languages = %v, want %v", got, want)
	}

	if got := specFixture().languageOptions("cli-tool"); len(got) != 0 {
		t.Errorf("languageOptions for an uninstalled project type = %v, want none", idsOf(got))
	}
}

func TestFrameworkOptionsFilteredByProjectTypeAndLanguage(t *testing.T) {
	goOpts := specFixture().frameworkOptions("backend-service", "go")
	if got, want := idsOf(goOpts), []string{"rest-api", "grpc"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("go frameworks = %v, want %v", got, want)
	}
	if goOpts[0].Name != "Go REST API Service" {
		t.Errorf("framework option name = %q, want the template displayName", goOpts[0].Name)
	}

	nodeOpts := specFixture().frameworkOptions("backend-service", "node")
	if got, want := idsOf(nodeOpts), []string{"http-api"}; len(got) != len(want) || got[0] != want[0] {
		t.Errorf("node frameworks = %v, want %v", got, want)
	}

	// The framework step is now filtered by language: go + node must not
	// offer the other's frameworks (V1's dead-end-combination limitation).
	if got := specFixture().frameworkOptions("backend-service", "python"); len(got) != 0 {
		t.Errorf("frameworkOptions for an uninstalled language = %v, want none", idsOf(got))
	}
}

func TestCapabilityOptionsUseDisplayName(t *testing.T) {
	got := specFixture().capabilityOptions()
	wantIDs := []string{"git-init", "readme"}
	for i, w := range wantIDs {
		if got[i].ID != w {
			t.Fatalf("capability %d: got %q, want %q", i, got[i].ID, w)
		}
	}
	if got[0].Name != "Initialize Git repository" {
		t.Errorf("capability option name = %q, want the plugin displayName", got[0].Name)
	}
}

func TestDefaultForPrefersThenFallsBack(t *testing.T) {
	opts := specFixture().projectTypeOptions()
	if got := specFixture().defaultFor(opts, "backend-service"); got != "backend-service" {
		t.Errorf("defaultFor with preferred installed: got %q, want the preferred id", got)
	}
	if got := specFixture().defaultFor(opts, "cli-tool"); got != "backend-service" {
		t.Errorf("defaultFor with uninstalled preferred: got %q, want the first option", got)
	}
	if got := (WizardSpec{}).defaultFor(nil, "go"); got != "" {
		t.Errorf("defaultFor with no options: got %q, want empty", got)
	}
}

func TestHumanizeID(t *testing.T) {
	if got := humanizeID("backend-service"); got != "Backend Service" {
		t.Errorf("humanizeID(\"backend-service\") = %q, want %q", got, "Backend Service")
	}
	if got := languageDisplayName("node"); got != "Node.js" {
		t.Errorf("languageDisplayName(\"node\") = %q, want %q", got, "Node.js")
	}
	if got := languageDisplayName("zig"); got != "Zig" {
		t.Errorf("languageDisplayName(\"zig\") = %q, want %q (humanized fallback)", got, "Zig")
	}
}
