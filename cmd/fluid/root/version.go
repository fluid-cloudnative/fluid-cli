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
	"fmt"

	"github.com/fluid-cloudnative/fluid-cli/cmd/fluid/internal/version"
	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of fluid",
		Long:  "Print version, git commit, and build date of fluid.",
		Example: `  # Print version
  fluid version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("fluid\n")
			fmt.Printf("  Version:    %s\n", version.Version)
			fmt.Printf("  Git Commit: %s\n", version.GitCommit)
			fmt.Printf("  Build Date: %s\n", version.BuildDate)
		},
	}
}
