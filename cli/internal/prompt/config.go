package prompt

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the CLI's small persisted local config — today, just the
// last-chosen theme. Only the interactive wizard path writes it (see
// ADR-0007): --theme/--answers runs never mutate persisted state, so
// scripted/CI usage stays side-effect-free.
type Config struct {
	Theme string `json:"theme,omitempty"`
}

// configPath returns the path to the CLI's config file, using the OS's
// standard per-user config directory (%AppData% on Windows,
// ~/Library/Application Support on macOS, $XDG_CONFIG_HOME or ~/.config
// on Linux).
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cli", "config.json"), nil
}

// LoadConfig reads the persisted config. A missing file is not an error
// — it returns a zero-value Config (no persisted theme yet).
func LoadConfig() (Config, error) {
	var c Config
	path, err := configPath()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

// SaveConfig writes the config, creating its directory if needed.
func SaveConfig(c Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
