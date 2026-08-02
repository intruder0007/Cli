package config

import (
	"strings"
	"testing"
)

func TestAnswersValidate(t *testing.T) {
	base := Answers{
		ProjectName: "demo", Theme: "default",
		ProjectType: "backend-service", Language: "go", Framework: "rest-api",
	}
	tests := []struct {
		name    string
		answers Answers
		wantErr string // substring of the expected error; "" = must pass
	}{
		{"valid", base, ""},
		{"empty theme defaults to default", Answers{ProjectName: "demo", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, ""},
		{"minimal theme is valid", Answers{ProjectName: "demo", Theme: "minimal", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, ""},
		{"missing project name", Answers{Theme: "default", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "project name is required"},
		{"name starting with a digit", Answers{ProjectName: "1demo", Theme: "default", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "must start with a letter"},
		{"name with a space", Answers{ProjectName: "my demo", Theme: "default", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "must start with a letter"},
		{"name with a slash", Answers{ProjectName: "a/b", Theme: "default", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "must start with a letter"},
		{"name with dots", Answers{ProjectName: "a.b", Theme: "default", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "must start with a letter"},
		{"unknown theme", Answers{ProjectName: "demo", Theme: "neon", ProjectType: "backend-service", Language: "go", Framework: "rest-api"}, "unknown theme"},
		{"missing project type", Answers{ProjectName: "demo", Theme: "default", Language: "go", Framework: "rest-api"}, "project type is required"},
		{"missing language", Answers{ProjectName: "demo", Theme: "default", ProjectType: "backend-service", Framework: "rest-api"}, "language is required"},
		{"missing framework", Answers{ProjectName: "demo", Theme: "default", ProjectType: "backend-service", Language: "go"}, "framework is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.answers.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() = %v, want nil error", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() = nil, want error containing %q", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
