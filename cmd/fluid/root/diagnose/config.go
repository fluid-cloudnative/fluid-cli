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
	"os"

	diagpkg "github.com/fluid-cloudnative/fluid-cli/pkg/diagnose"
	tuicommon "github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
	tuiconfig "github.com/fluid-cloudnative/fluid-cli/pkg/tui/diagnoseconfig"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage diagnose AI/LLM settings",
		Long: `Manage Fluid diagnose AI settings stored in ~/.fluid/config.

Run without subcommands to open an interactive setup form (Bubble Tea TUI).
Subcommands (set, get, unset, view) are available for scripting.

Settings are used for OpenAI-compatible LLM analysis during fluid diagnose.
Prefer FLUID_LLM_API_KEY for secrets instead of storing apiKey in the config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown argument %q (use subcommands or run without args for interactive config)", args[0])
			}
			if err := tuicommon.EnsureInteractive(cmd.InOrStdin(), cmd.OutOrStdout(), "diagnose config"); err != nil {
				return err
			}
			return tuiconfig.Run(cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}

	cmd.AddCommand(
		newConfigSetCommand(),
		newConfigGetCommand(),
		newConfigUnsetCommand(),
		newConfigViewCommand(),
	)

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a diagnose LLM configuration value",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "llm-endpoint <url>",
			Short: "Set the default LLM API base URL",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.SetLLMEndpoint(args[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: saved llm endpoint\n")
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-api-key <key>",
			Short: "Set the LLM API key in ~/.fluid/config (prefer FLUID_LLM_API_KEY env)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.SetLLMAPIKey(args[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: saved llm api key\n")
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-model <model>",
			Short: "Set the default LLM model name",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.SetLLMModel(args[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: saved llm model\n")
				return nil
			},
		},
	)
	return cmd
}

func newConfigGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Print a diagnose LLM configuration value",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "llm-endpoint",
			Short: "Print the configured LLM API endpoint",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				endpoint, err := diagpkg.GetLLMEndpoint()
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), endpoint)
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-api-key",
			Short: "Print whether an LLM API key is configured",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				ok, err := diagpkg.GetLLMAPIKeyConfigured()
				if err != nil {
					return err
				}
				if ok {
					fmt.Fprintln(cmd.OutOrStdout(), "configured")
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-model",
			Short: "Print the configured LLM model",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				model, err := diagpkg.GetLLMModel()
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), model)
				return nil
			},
		},
	)
	return cmd
}

func newConfigUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset",
		Short: "Remove a diagnose LLM configuration value",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "llm-endpoint",
			Short: "Remove the configured LLM API endpoint",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.UnsetLLMEndpoint(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: removed llm endpoint\n")
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-api-key",
			Short: "Remove the LLM API key from ~/.fluid/config",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.UnsetLLMAPIKey(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: removed llm api key\n")
				return nil
			},
		},
		&cobra.Command{
			Use:   "llm-model",
			Short: "Remove the configured LLM model",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := diagpkg.UnsetLLMModel(); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "diagnose config: removed llm model\n")
				return nil
			},
		},
	)
	return cmd
}

func newConfigViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Show the diagnose section of ~/.fluid/config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := diagpkg.LoadUserConfig()
			if err != nil {
				return err
			}
			if cfg.Diagnose.LLM.APIKey != "" {
				cfg.Diagnose.LLM.APIKey = "<set>"
			}
			path, err := diagpkg.ConfigPath()
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Fprintf(cmd.OutOrStdout(), "# %s (not created yet)\n", path)
				return nil
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "# %s\n%s", path, string(data))
			return nil
		},
	}
	return cmd
}
