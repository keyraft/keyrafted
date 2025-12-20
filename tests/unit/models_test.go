package unit

import (
	"keyrafted/internal/models"
	"testing"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{"valid single level", "project", false},
		{"valid two levels", "project/prod", false},
		{"valid three levels", "project/prod/api", false},
		{"valid with underscores", "my_project/prod_env", false},
		{"valid with hyphens", "my-project/prod-env", false},
		{"empty namespace", "", true},
		{"invalid characters", "project/prod@api", true},
		{"too long", string(make([]byte, 257)), true},
		{"too many slashes", "a/b/c/d", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateNamespace(tt.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNamespace() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid alphanumeric", "DATABASE_URL", false},
		{"valid with dots", "app.config.timeout", false},
		{"valid with hyphens", "app-config-timeout", false},
		{"valid with underscores", "app_config_timeout", false},
		{"empty key", "", true},
		{"invalid characters", "key@value", true},
		{"too long", string(make([]byte, 257)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
