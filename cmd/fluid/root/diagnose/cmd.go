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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	diagpkg "github.com/fluid-cloudnative/fluid-cli/pkg/diagnose"
	fluidscheme "github.com/fluid-cloudnative/fluid-cli/pkg/scheme"
	tuicommon "github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
	tuidiagnose "github.com/fluid-cloudnative/fluid-cli/pkg/tui/diagnose"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options holds the options for the diagnose command.
type Options struct {
	configFlags *genericclioptions.ConfigFlags

	namespace   string
	datasetName string
	output      string

	// packaging options
	archive   bool
	outputDir string

	// collection options
	noLogs                bool
	includeControllerLogs bool
	since                 string

	promptFile  string
	llmEndpoint string
	llmModel    string
	llmSkip     bool
}

func NewDiagnoseCommand(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &Options{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "diagnose <dataset-name>",
		Short: "Collect diagnostic data for a Fluid Dataset and its Runtime(s)",
		Long: `Diagnose collects diagnostic data for a given Fluid Dataset, including:
  - Dataset and Runtime CR YAML
  - Pod describe output and logs
  - Namespace events
  - PVC and PV descriptions
  - Optional Fluid controller logs

All artifacts are written to a timestamped directory and optionally packaged
into a tar.gz archive.`,
		Example: `  # Diagnose a Dataset in the current namespace
  fluid diagnose my-dataset

  # Diagnose and produce a tar.gz archive
  fluid diagnose my-dataset -n default --archive

  # Write artifacts to a custom directory without collecting logs
  fluid diagnose my-dataset -n default --output-dir /tmp/diag --no-logs

  # Only collect events from the last hour
  fluid diagnose my-dataset -n default --since 1h

  # Configure LLM settings (OpenAI-compatible API)
  fluid diagnose config set llm-endpoint https://api.openai.com/v1
  export FLUID_LLM_API_KEY=sk-...

  # Collect artifacts and request LLM analysis
  fluid diagnose my-dataset -n default -o dir`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.datasetName = args[0]
			if ns, err := cmd.Flags().GetString("namespace"); err == nil && ns != "" {
				o.namespace = ns
			}
			if o.namespace == "" {
				ns, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
				if err == nil {
					o.namespace = ns
				}
			}
			if o.namespace == "" {
				o.namespace = "default"
			}
			return o.run(cmd)
		},
	}

	// packaging
	cmd.Flags().StringVarP(&o.output, "output", "o", "tui", "Output mode: tui|dir|stdout")
	cmd.Flags().BoolVar(&o.archive, "archive", false, "Package artifacts into a tar.gz archive")
	cmd.Flags().StringVar(&o.outputDir, "output-dir", "", "Directory to write artifacts (default: fluid-diagnose-<dataset>-<timestamp>)")

	// collection
	cmd.Flags().BoolVar(&o.noLogs, "no-logs", false, "Skip collecting pod logs (useful in large clusters)")
	cmd.Flags().BoolVar(&o.includeControllerLogs, "include-controller-logs", false, "Also collect Fluid controller logs from fluid-system namespace")
	cmd.Flags().StringVar(&o.since, "since", "", "Only collect logs/events newer than this duration (e.g. 1h, 30m)")

	// AI-assisted diagnosis
	cmd.Flags().StringVar(&o.promptFile, "prompt-file", "", "Also write prompt-ready diagnostic text to this file")
	cmd.Flags().StringVar(&o.llmEndpoint, "llm-endpoint", "", "LLM API base URL (overrides FLUID_LLM_ENDPOINT and ~/.fluid/config)")
	cmd.Flags().StringVar(&o.llmModel, "llm-model", "", "LLM model name (overrides FLUID_LLM_MODEL and ~/.fluid/config)")
	cmd.Flags().BoolVar(&o.llmSkip, "llm-skip", false, "Skip LLM analysis (when endpoint is configured, analysis runs by default)")

	cmd.AddCommand(newConfigCommand())

	return cmd
}

func (o *Options) run(cmd *cobra.Command) error {
	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("building REST config: %w", err)
	}

	c, err := client.New(restConfig, client.Options{Scheme: fluidscheme.Scheme})
	if err != nil {
		return fmt.Errorf("creating Kubernetes client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("creating Kubernetes typed client: %w", err)
	}

	runner := diagpkg.NewRunner(c, kubeClient)
	if o.output != "tui" && o.output != "dir" && o.output != "stdout" {
		return fmt.Errorf("invalid output mode %q, expected tui|dir|stdout", o.output)
	}

	llmSettings, err := diagpkg.ResolveLLMSettings(o.llmEndpoint, o.llmModel, o.llmSkip, cmd.Flags().Changed("llm-skip"))
	if err != nil {
		return err
	}

	runOpts := diagpkg.Options{
		DatasetName:           o.datasetName,
		Namespace:             o.namespace,
		Output:                o.output,
		Archive:               o.archive,
		OutputDir:             o.outputDir,
		NoLogs:                o.noLogs,
		IncludeControllerLogs: o.includeControllerLogs,
		Since:                 o.since,
		PromptFile:            o.promptFile,
		LLMEndpoint:           llmSettings.Endpoint,
		LLMAPIKey:             llmSettings.APIKey,
		LLMModel:              llmSettings.Model,
		LLMSkip:               llmSettings.Skip,
		Stderr:                cmd.ErrOrStderr(),
	}
	if o.output == "tui" {
		runOpts.Output = "dir"
	}

	result, err := runner.Run(cmd.Context(), runOpts)
	if err != nil {
		if strings.Contains(err.Error(), "no kind is registered") ||
			strings.Contains(err.Error(), "no matches for kind") ||
			strings.Contains(err.Error(), "no matches for") ||
			strings.Contains(err.Error(), "unable to retrieve the complete list of server APIs") {
			return fmt.Errorf("Fluid CRDs are not installed on this cluster (data.fluid.io/v1alpha1 not found).\n"+
				"Install Fluid first: https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md\n"+
				"Original error: %w", err)
		}
		return err
	}

	if o.output == "tui" {
		if err := tuicommon.EnsureInteractive(cmd.InOrStdin(), cmd.OutOrStdout(), "diagnose --output=tui"); err != nil {
			return err
		}
		viewData, err := buildTUIData(o.datasetName, o.namespace, result)
		if err != nil {
			return err
		}
		return tuidiagnose.Run(cmd.InOrStdin(), cmd.OutOrStdout(), viewData)
	}

	if result.Stdout != "" {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), result.Stdout)
	}
	if o.output == "stdout" {
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "diagnose: collected artifacts at %s\n", result.OutputPath)
	if result.ContextPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "diagnose: diagnostic context written to %s\n", result.ContextPath)
	}
	if result.PromptPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "diagnose: prompt written to %s\n", result.PromptPath)
	}
	if result.LLMAnalysisPath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "diagnose: LLM analysis written to %s\n", result.LLMAnalysisPath)
	}
	if result.ArchivePath != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "diagnose: archive written to %s\n", result.ArchivePath)
	}
	if result.PartialFailureCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "diagnose: completed with %d partial collection issues (see summary.txt/manifest.json)\n", result.PartialFailureCount)
	}
	return nil
}

type manifest struct {
	Artifacts []manifestEntry `json:"artifacts"`
}

type manifestEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func buildTUIData(datasetName, namespace string, result *diagpkg.Result) (tuidiagnose.ViewData, error) {
	summaryPath := filepath.Join(result.OutputPath, "summary.txt")
	summaryBytes, err := os.ReadFile(summaryPath)
	if err != nil {
		return tuidiagnose.ViewData{}, fmt.Errorf("reading diagnose summary: %w", err)
	}

	manifestPath := filepath.Join(result.OutputPath, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return tuidiagnose.ViewData{}, fmt.Errorf("reading diagnose manifest: %w", err)
	}
	var mf manifest
	if err := json.Unmarshal(manifestBytes, &mf); err != nil {
		return tuidiagnose.ViewData{}, fmt.Errorf("decoding diagnose manifest: %w", err)
	}

	warnings := make([]string, 0)
	for _, line := range strings.Split(string(summaryBytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			warnings = append(warnings, strings.TrimPrefix(trimmed, "- "))
		}
	}

	artifacts := make([]tuidiagnose.ManifestEntry, 0, len(mf.Artifacts))
	for _, entry := range mf.Artifacts {
		artifacts = append(artifacts, tuidiagnose.ManifestEntry{
			Path:   entry.Path,
			Status: entry.Status,
			Reason: entry.Reason,
		})
	}

	return tuidiagnose.ViewData{
		Dataset:             datasetName,
		Namespace:           namespace,
		OutputPath:          result.OutputPath,
		ArchivePath:         result.ArchivePath,
		PartialFailureCount: result.PartialFailureCount,
		Summary:             string(summaryBytes),
		Artifacts:           artifacts,
		WarningEvents:       warnings,
	}, nil
}
