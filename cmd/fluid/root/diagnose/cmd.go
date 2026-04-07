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

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Options holds the options for the diagnose command.
type Options struct {
	configFlags *genericclioptions.ConfigFlags

	namespace   string
	datasetName string

	// packaging options
	archive   bool
	outputDir string

	// collection options
	noLogs                bool
	includeControllerLogs bool
	since                 string

	// LLM/AI output (Phase 3 stub)
	promptFile string
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
  fluid diagnose my-dataset -n default --since 1h`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.datasetName = args[0]
			if ns, err := cmd.Flags().GetString("namespace"); err == nil && ns != "" {
				o.namespace = ns
			}
			return o.run(cmd)
		},
	}

	// packaging
	cmd.Flags().BoolVar(&o.archive, "archive", false, "Package artifacts into a tar.gz archive")
	cmd.Flags().StringVar(&o.outputDir, "output-dir", "", "Directory to write artifacts (default: fluid-diagnose-<dataset>-<timestamp>)")

	// collection
	cmd.Flags().BoolVar(&o.noLogs, "no-logs", false, "Skip collecting pod logs (useful in large clusters)")
	cmd.Flags().BoolVar(&o.includeControllerLogs, "include-controller-logs", false, "Also collect Fluid controller logs from fluid-system namespace")
	cmd.Flags().StringVar(&o.since, "since", "", "Only collect logs/events newer than this duration (e.g. 1h, 30m)")

	// Phase 3 stub
	cmd.Flags().StringVar(&o.promptFile, "prompt-file", "", "[Phase 3] Write prompt-ready diagnostic context to this file")
	_ = cmd.Flags().MarkHidden("prompt-file")

	return cmd
}

func (o *Options) run(cmd *cobra.Command) error {
	// Phase 0 placeholder — full implementation in Phase 2.
	fmt.Fprintf(cmd.OutOrStdout(), "diagnose: Dataset=%q Namespace=%q (not yet implemented — Phase 2)\n",
		o.datasetName, o.namespace)
	fmt.Fprintf(cmd.OutOrStdout(), "  flags: archive=%v output-dir=%q no-logs=%v since=%q\n",
		o.archive, o.outputDir, o.noLogs, o.since)
	return nil
}
