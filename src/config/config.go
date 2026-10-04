package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config defines settings for rig.
type Config struct {
	Endpoint           string   `json:"endpoint"`
	Model              string   `json:"model"`
	MaxContextTokens   int      `json:"max_context_tokens"`
	MaxSteps           int      `json:"max_steps"`
	InstructionsFile   string   `json:"instructions_file"`
	ContextFiles       []string `json:"context_files"`
	MaxToolOutputChars int      `json:"max_tool_output_chars"`
}

// Default returns sensible baseline settings.
func Default() Config {
	return Config{
		Endpoint:           "http://127.0.0.1:1234/v1",
		Model:              "",
		MaxContextTokens:   16384,
		MaxSteps:           10,
		InstructionsFile:   "instructions.md",
		ContextFiles:       []string{},
		MaxToolOutputChars: 3000,
	}
}

// Load reads config.json from path, or returns default if not found.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil && os.IsNotExist(err) && path == "config.json" {
		// Fallback check in config/ subdirectory
		data, err = os.ReadFile("config/config.json")
	}

	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Not an error if config doesn't exist yet
		}
		return cfg, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config JSON from '%s': %w", path, err)
	}

	return cfg, nil
}
