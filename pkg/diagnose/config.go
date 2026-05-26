// Copyright 2026 The Fluid Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnose

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	envLLMEndpoint = "FLUID_LLM_ENDPOINT"
	envLLMAPIKey   = "FLUID_LLM_API_KEY"
	envLLMModel    = "FLUID_LLM_MODEL"
)

// UserConfig is the on-disk Fluid CLI configuration (~/.fluid/config).
type UserConfig struct {
	Diagnose DiagnoseConfig `json:"diagnose" yaml:"diagnose"`
}

type DiagnoseConfig struct {
	LLM LLMConfig `json:"llm" yaml:"llm"`
}

type LLMConfig struct {
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	APIKey   string `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	Model    string `json:"model,omitempty" yaml:"model,omitempty"`
}

// LLMSettings are resolved settings used at diagnose runtime.
type LLMSettings struct {
	Endpoint string
	APIKey   string
	Model    string
	Skip     bool
}

// ConfigPath returns the default user config file path (~/.fluid/config).
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".fluid", "config"), nil
}

// LoadUserConfig reads ~/.fluid/config. A missing file yields an empty config.
func LoadUserConfig() (UserConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return UserConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return UserConfig{}, nil
		}
		return UserConfig{}, fmt.Errorf("reading config %q: %w", path, err)
	}
	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return UserConfig{}, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return cfg, nil
}

// SaveUserConfig writes ~/.fluid/config with mode 0600.
func SaveUserConfig(cfg UserConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config %q: %w", path, err)
	}
	return nil
}

// ValidateLLMEndpoint checks that endpoint is a valid HTTP(S) URL with a host.
func ValidateLLMEndpoint(endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("endpoint must use http or https scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("endpoint must include a host")
	}
	return nil
}

// ResolveLLMSettings resolves LLM settings with precedence:
// endpoint: flag > FLUID_LLM_ENDPOINT > ~/.fluid/config
// api_key:  FLUID_LLM_API_KEY > ~/.fluid/config
// model:    flag > FLUID_LLM_MODEL > ~/.fluid/config > defaultLLMModel
// skip:     explicit --llm-skip; when endpoint is set and --llm-skip not passed, LLM is enabled
func ResolveLLMSettings(flagEndpoint, flagModel string, flagSkip, flagSkipSet bool) (LLMSettings, error) {
	settings := LLMSettings{Skip: true, Model: defaultLLMModel}

	cfg, err := LoadUserConfig()
	if err != nil {
		return LLMSettings{}, err
	}

	flagEndpoint = strings.TrimSpace(flagEndpoint)
	if flagEndpoint != "" {
		if err := ValidateLLMEndpoint(flagEndpoint); err != nil {
			return LLMSettings{}, err
		}
		settings.Endpoint = flagEndpoint
	} else if envEndpoint := strings.TrimSpace(os.Getenv(envLLMEndpoint)); envEndpoint != "" {
		if err := ValidateLLMEndpoint(envEndpoint); err != nil {
			return LLMSettings{}, fmt.Errorf("%s: %w", envLLMEndpoint, err)
		}
		settings.Endpoint = envEndpoint
	} else if fileEndpoint := strings.TrimSpace(cfg.Diagnose.LLM.Endpoint); fileEndpoint != "" {
		if err := ValidateLLMEndpoint(fileEndpoint); err != nil {
			return LLMSettings{}, fmt.Errorf("config file endpoint: %w", err)
		}
		settings.Endpoint = fileEndpoint
	}

	if envKey := strings.TrimSpace(os.Getenv(envLLMAPIKey)); envKey != "" {
		settings.APIKey = envKey
	} else if fileKey := strings.TrimSpace(cfg.Diagnose.LLM.APIKey); fileKey != "" {
		settings.APIKey = fileKey
	}

	flagModel = strings.TrimSpace(flagModel)
	if flagModel != "" {
		settings.Model = flagModel
	} else if envModel := strings.TrimSpace(os.Getenv(envLLMModel)); envModel != "" {
		settings.Model = envModel
	} else if fileModel := strings.TrimSpace(cfg.Diagnose.LLM.Model); fileModel != "" {
		settings.Model = fileModel
	}

	if settings.Endpoint != "" {
		if flagSkipSet {
			settings.Skip = flagSkip
		} else {
			settings.Skip = false
		}
	}

	if !settings.Skip && settings.Endpoint != "" && settings.APIKey == "" {
		return LLMSettings{}, fmt.Errorf("LLM API key is required when LLM analysis is enabled (set %s or `fluid diagnose config set llm-api-key`)", envLLMAPIKey)
	}

	return settings, nil
}

// SetLLMEndpoint persists the LLM endpoint in user config.
func SetLLMEndpoint(endpoint string) error {
	if err := ValidateLLMEndpoint(endpoint); err != nil {
		return err
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.Endpoint = strings.TrimSpace(endpoint)
	return SaveUserConfig(cfg)
}

// GetLLMEndpoint returns the configured LLM endpoint, or empty if unset.
func GetLLMEndpoint() (string, error) {
	cfg, err := LoadUserConfig()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.Diagnose.LLM.Endpoint), nil
}

// UnsetLLMEndpoint removes the LLM endpoint from user config.
func UnsetLLMEndpoint() error {
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.Endpoint = ""
	return SaveUserConfig(cfg)
}

// SetLLMAPIKey persists the LLM API key in user config.
func SetLLMAPIKey(apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("api key is required")
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.APIKey = apiKey
	return SaveUserConfig(cfg)
}

// GetLLMAPIKey returns whether an API key is configured (never returns the secret value).
func GetLLMAPIKeyConfigured() (bool, error) {
	if strings.TrimSpace(os.Getenv(envLLMAPIKey)) != "" {
		return true, nil
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(cfg.Diagnose.LLM.APIKey) != "", nil
}

// UnsetLLMAPIKey removes the LLM API key from user config.
func UnsetLLMAPIKey() error {
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.APIKey = ""
	return SaveUserConfig(cfg)
}

// SetLLMModel persists the default LLM model name.
func SetLLMModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Errorf("model is required")
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.Model = model
	return SaveUserConfig(cfg)
}

// GetLLMModel returns the configured model or empty if unset.
func GetLLMModel() (string, error) {
	cfg, err := LoadUserConfig()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.Diagnose.LLM.Model), nil
}

// UnsetLLMModel removes the configured model from user config.
func UnsetLLMModel() error {
	cfg, err := LoadUserConfig()
	if err != nil {
		return err
	}
	cfg.Diagnose.LLM.Model = ""
	return SaveUserConfig(cfg)
}

// LLMFormDefaults are initial values for the interactive config form.
type LLMFormDefaults struct {
	Endpoint     string
	Model        string
	APIKeyInFile bool
	APIKeyInEnv  bool
	ConfigPath   string
}

// LoadLLMFormDefaults loads current file-based LLM settings for the config TUI.
func LoadLLMFormDefaults() (LLMFormDefaults, error) {
	path, err := ConfigPath()
	if err != nil {
		return LLMFormDefaults{}, err
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		return LLMFormDefaults{}, err
	}
	return LLMFormDefaults{
		Endpoint:     strings.TrimSpace(cfg.Diagnose.LLM.Endpoint),
		Model:        strings.TrimSpace(cfg.Diagnose.LLM.Model),
		APIKeyInFile: strings.TrimSpace(cfg.Diagnose.LLM.APIKey) != "",
		APIKeyInEnv:  strings.TrimSpace(os.Getenv(envLLMAPIKey)) != "",
		ConfigPath:   path,
	}, nil
}

// LLMFormValues are submitted values from the interactive config form.
type LLMFormValues struct {
	Endpoint string
	APIKey   string
	Model    string
}

// ApplyLLMFormValues persists form values to ~/.fluid/config.
// An empty API key leaves the existing key unchanged.
func ApplyLLMFormValues(v LLMFormValues) error {
	endpoint := strings.TrimSpace(v.Endpoint)
	if endpoint == "" {
		if err := UnsetLLMEndpoint(); err != nil {
			return err
		}
	} else if err := SetLLMEndpoint(endpoint); err != nil {
		return err
	}

	if key := strings.TrimSpace(v.APIKey); key != "" {
		if err := SetLLMAPIKey(key); err != nil {
			return err
		}
	}

	model := strings.TrimSpace(v.Model)
	if model == "" {
		if err := UnsetLLMModel(); err != nil {
			return err
		}
	} else if err := SetLLMModel(model); err != nil {
		return err
	}

	return nil
}
