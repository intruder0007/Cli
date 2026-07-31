package sdk

import "testing"

func validTemplateManifest() Manifest {
	return Manifest{
		ProtocolVersion: "1", Name: "t", Version: "0.1.0", Kind: "template",
		ProjectType: "backend-service", Language: "go", Framework: "rest-api",
		Entrypoint: "./t",
	}
}

func validCapabilityManifest() Manifest {
	return Manifest{
		ProtocolVersion: "1", Name: "c", Version: "0.1.0", Kind: "capability",
		CapabilityID: "c", Entrypoint: "./c",
	}
}

func TestValidateValidManifests(t *testing.T) {
	if err := validTemplateManifest().Validate(); err != nil {
		t.Errorf("valid template manifest should pass Validate(), got: %v", err)
	}
	if err := validCapabilityManifest().Validate(); err != nil {
		t.Errorf("valid capability manifest should pass Validate(), got: %v", err)
	}
}

func TestValidateMissingCommonFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"protocolVersion", func(m *Manifest) { m.ProtocolVersion = "" }},
		{"name", func(m *Manifest) { m.Name = "" }},
		{"version", func(m *Manifest) { m.Version = "" }},
		{"entrypoint", func(m *Manifest) { m.Entrypoint = "" }},
	}
	for _, c := range cases {
		t.Run("missing "+c.name, func(t *testing.T) {
			m := validTemplateManifest()
			c.mutate(&m)
			if err := m.Validate(); err == nil {
				t.Errorf("Validate() with missing %s: want error, got nil", c.name)
			}
		})
	}
}

func TestValidateKindSpecificFields(t *testing.T) {
	t.Run("template missing projectType/language/framework", func(t *testing.T) {
		m := validTemplateManifest()
		m.ProjectType = ""
		if err := m.Validate(); err == nil {
			t.Error("template missing projectType: want error, got nil")
		}
	})
	t.Run("capability missing capabilityId", func(t *testing.T) {
		m := validCapabilityManifest()
		m.CapabilityID = ""
		if err := m.Validate(); err == nil {
			t.Error("capability missing capabilityId: want error, got nil")
		}
	})
	t.Run("unknown kind", func(t *testing.T) {
		m := validTemplateManifest()
		m.Kind = "bogus"
		if err := m.Validate(); err == nil {
			t.Error("unknown kind: want error, got nil")
		}
	})
}
