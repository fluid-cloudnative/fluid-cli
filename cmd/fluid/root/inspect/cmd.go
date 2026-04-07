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
	"strings"

	"github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	fluidscheme "github.com/fluid-cloudnative/fluid-cli/pkg/scheme"
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
		Use:   "inspect <dataset-name>",
		Short: "List all Kubernetes resources associated with a Fluid Dataset",
		Long: `Inspect lists all Kubernetes resources (Pods, StatefulSets, DaemonSets,
PVCs, PVs, Services, etc.) owned by a given Fluid Dataset and its Runtime(s),
along with their current status.`,
		Example: `  # Inspect a Dataset in the default namespace
  fluid inspect my-dataset

  # Inspect a Dataset in a specific namespace
  fluid inspect my-dataset -n default

  # Output as JSON
  fluid inspect my-dataset -n default -o json

  # Show extra columns (node, restarts)
  fluid inspect my-dataset -n default --wide`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.datasetName = args[0]

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

	cmd.Flags().StringVarP(&o.outputFormat, "output", "o", "table", "Output format: table|json|yaml")
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

	inspector := inspect.New(c)
	report, err := inspector.Run(context.Background(), o.datasetName, o.namespace)
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
		return err
	}

	return inspect.Print(cmd.OutOrStdout(), report, o.outputFormat, o.wide)
}
