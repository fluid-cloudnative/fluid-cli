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

package root

import (
	"github.com/fluid-cloudnative/fluid-cli/cmd/fluid/root/diagnose"
	"github.com/fluid-cloudnative/fluid-cli/cmd/fluid/root/inspect"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewRootCmd creates the root command for fluid.
func NewRootCmd() *cobra.Command {
	configFlags := genericclioptions.NewConfigFlags(true)

	root := &cobra.Command{
		Use:   "fluid",
		Short: "Inspect and diagnose Fluid-managed datasets",
		Long: `fluid is a CLI for the Fluid project.

It provides commands to inspect and diagnose Fluid-managed Datasets and their
associated Kubernetes resources (Runtimes, Pods, PVCs, PVs, Services, etc.).

Install: copy the binary to a directory in your PATH as 'fluid'.
Usage:   fluid <subcommand>`,
		Example: `  # List resources owned by a Dataset
  fluid inspect my-dataset -n default

  # Collect diagnostic data and archive it
  fluid diagnose my-dataset -n default --archive

  # Print version
  fluid version`,
		SilenceUsage: true,
	}

	// Bind standard Kubernetes config flags (--kubeconfig, --context,
	// --cluster, --user, --as, --as-group, --insecure-skip-tls-verify,
	// --request-timeout, --server, --tls-server-name, --token,
	// --certificate-authority, --client-certificate, --client-key).
	configFlags.AddFlags(root.PersistentFlags())

	// Group commands for clean --help output.
	root.AddGroup(
		&cobra.Group{ID: "inspect", Title: "Inspect:"},
		&cobra.Group{ID: "diagnose", Title: "Diagnose:"},
		&cobra.Group{ID: "utils", Title: "Utility:"},
	)

	addToGroup(root, "inspect", inspect.NewInspectCommand(configFlags))
	addToGroup(root, "diagnose", diagnose.NewDiagnoseCommand(configFlags))
	addToGroup(root, "utils", versionCommand())

	return root
}

func addToGroup(root *cobra.Command, groupID string, cmd *cobra.Command) {
	cmd.GroupID = groupID
	root.AddCommand(cmd)
}
