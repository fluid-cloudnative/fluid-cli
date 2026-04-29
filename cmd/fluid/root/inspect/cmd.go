// Copyright 2025 The Fluid Authors.
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

package inspect

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	fluidscheme "github.com/fluid-cloudnative/fluid-cli/pkg/scheme"
	tuicommon "github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
	tuidatasetselect "github.com/fluid-cloudnative/fluid-cli/pkg/tui/datasetselect"
	tuiinspect "github.com/fluid-cloudnative/fluid-cli/pkg/tui/inspect"
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options holds the options for the inspect command.
type Options struct {
	configFlags *genericclioptions.ConfigFlags

	outputFormat string
	wide         bool

	namespace   string
	datasetName string
}

func NewInspectCommand(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &Options{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "inspect [dataset-name]",
		Short: "List all Kubernetes resources associated with a Fluid Dataset",
		Long: `Inspect lists all Kubernetes resources (Pods, StatefulSets, DaemonSets,
PVCs, PVs, Services, etc.) owned by a given Fluid Dataset and its Runtime(s),
along with their current status.`,
		Example: `  # Inspect a Dataset in the default namespace
  fluid inspect my-dataset

  # Launch a TUI selector to choose a Dataset
  fluid inspect

  # Inspect a Dataset in a specific namespace
  fluid inspect my-dataset -n default

  # Output as JSON
  fluid inspect my-dataset -n default -o json

  # Show extra columns (node, restarts)
  fluid inspect my-dataset -n default --wide`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.datasetName = args[0]
			}

			// Prefer the -n flag set on this command; fall back to configFlags namespace.
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

	cmd.Flags().StringVarP(&o.outputFormat, "output", "o", "tui", "Output format: tui|table|json|yaml")
	cmd.Flags().BoolVar(&o.wide, "wide", false, "Show additional columns (node, restarts)")

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

	if o.datasetName == "" {
		if err := tuicommon.EnsureInteractive(cmd.InOrStdin(), cmd.OutOrStdout(), "inspect dataset selector"); err != nil {
			return fmt.Errorf("dataset name is required in non-interactive mode: %w", err)
		}
		names, err := listDatasetNames(cmd.Context(), c, o.namespace)
		if err != nil {
			return err
		}
		o.datasetName, err = tuidatasetselect.Run(
			cmd.InOrStdin(),
			cmd.OutOrStdout(),
			names,
			o.namespace,
		)
		if err != nil {
			return err
		}
	}

	inspector := inspect.New(c)
	report, err := inspector.Run(cmd.Context(), o.datasetName, o.namespace)
	if err != nil {
		// Give a clear hint when Fluid CRDs are not installed on the cluster.
		if strings.Contains(err.Error(), "no kind is registered") ||
			strings.Contains(err.Error(), "no matches for kind") ||
			strings.Contains(err.Error(), "no matches for") ||
			strings.Contains(err.Error(), "unable to retrieve the complete list of server APIs") {
			return fmt.Errorf("Fluid CRDs are not installed on this cluster (data.fluid.io/v1alpha1 not found).\n"+
				"Install Fluid first: https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md\n"+
				"Original error: %w", err)
		}
		if isClusterConnectivityError(err) {
			return fmt.Errorf("cannot reach Kubernetes API server while inspecting dataset %q in namespace %q.\n"+
				"Verify kubeconfig/context and cluster health (try: kubectl cluster-info).\n"+
				"Original error: %w", o.datasetName, o.namespace, err)
		}
		return err
	}

	return renderInspectOutput(cmd, report, o.outputFormat, o.wide)
}

func renderInspectOutput(cmd *cobra.Command, report *inspect.DatasetReport, outputFormat string, wide bool) error {
	switch outputFormat {
	case "tui":
		if err := tuicommon.EnsureInteractive(cmd.InOrStdin(), cmd.OutOrStdout(), "inspect --output=tui"); err != nil {
			return err
		}
		return tuiinspect.Run(cmd.InOrStdin(), cmd.OutOrStdout(), report, wide)
	case "table", "json", "yaml":
		return inspect.Print(cmd.OutOrStdout(), report, outputFormat, wide)
	default:
		return fmt.Errorf("invalid output format %q, expected tui|table|json|yaml", outputFormat)
	}
}

func listDatasetNames(ctx context.Context, c client.Client, namespace string) ([]string, error) {
	var datasets fluidv1alpha1.DatasetList
	if err := c.List(ctx, &datasets, client.InNamespace(namespace)); err != nil {
		if strings.Contains(err.Error(), "no kind is registered") ||
			strings.Contains(err.Error(), "no matches for kind") ||
			strings.Contains(err.Error(), "no matches for") ||
			strings.Contains(err.Error(), "unable to retrieve the complete list of server APIs") {
			return nil, fmt.Errorf("Fluid CRDs are not installed on this cluster (data.fluid.io/v1alpha1 not found).\n"+
				"Install Fluid first: https://github.com/fluid-cloudnative/fluid/blob/master/docs/en/userguide/install.md\n"+
				"Original error: %w", err)
		}
		if isClusterConnectivityError(err) {
			return nil, fmt.Errorf("cannot reach Kubernetes API server while listing datasets in namespace %q.\n"+
				"Verify kubeconfig/context and cluster health (try: kubectl cluster-info).\n"+
				"Original error: %w", namespace, err)
		}
		return nil, fmt.Errorf("listing datasets in namespace %q: %w", namespace, err)
	}
	if len(datasets.Items) == 0 {
		return nil, fmt.Errorf("no datasets found in namespace %q", namespace)
	}

	names := make([]string, 0, len(datasets.Items))
	for _, ds := range datasets.Items {
		names = append(names, ds.Name)
	}
	sort.Strings(names)
	return names, nil
}

func isClusterConnectivityError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"tls handshake timeout",
		"i/o timeout",
		"connection refused",
		"no such host",
		"context deadline exceeded",
		"unable to connect to the server",
	}
	for _, needle := range needles {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
