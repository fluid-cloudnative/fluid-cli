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
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"sigs.k8s.io/yaml"
)

// Print renders the DatasetReport to w in the requested format.
// format is one of: "table" (default), "json", "yaml".
// wide enables additional columns (Node, Restarts) in resource tables.
func Print(w io.Writer, report *DatasetReport, format string, wide bool) error {
	switch format {
	case "", "table":
		return printTable(w, report, wide)
	case "json":
		return printJSON(w, report)
	case "yaml":
		return printYAML(w, report)
	default:
		return fmt.Errorf("unsupported output format %q (supported: table, json, yaml)", format)
	}
}

func printJSON(w io.Writer, report *DatasetReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func printYAML(w io.Writer, report *DatasetReport) error {
	data, err := yaml.Marshal(report)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func printTable(w io.Writer, report *DatasetReport, wide bool) error {
	if err := printIdentitySection(w, report); err != nil {
		return err
	}
	if err := printSpecSection(w, report); err != nil {
		return err
	}
	if err := printStatusSection(w, report); err != nil {
		return err
	}
	if err := printResourcesSection(w, report, wide); err != nil {
		return err
	}
	return nil
}

// printIdentitySection prints the Dataset identity and metadata.
//
// Example:
//
//	Name:         test
//	Namespace:    default
//	Labels:       fluid.io/dataset-id=default-test
//	API Version:  data.fluid.io/v1alpha1
//	Kind:         Dataset
//	Metadata:
//	  Creation Timestamp:  2026-03-26T02:28:09Z
//	  Generation:          1
//	  Resource Version:    4591
//	  UID:                 f7aa137a-b796-44b6-8de5-7a056fba2b1a
//	  Finalizers:
//	    fluid-dataset-controller-finalizer
func printIdentitySection(w io.Writer, report *DatasetReport) error {
	id := report.Identity
	fmt.Fprintf(w, "Name:         %s\n", id.Name)
	fmt.Fprintf(w, "Namespace:    %s\n", id.Namespace)

	// Labels – sorted for determinism
	if len(id.Labels) == 0 {
		fmt.Fprintf(w, "Labels:       <none>\n")
	} else {
		keys := make([]string, 0, len(id.Labels))
		for k := range id.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, id.Labels[k]))
		}
		fmt.Fprintf(w, "Labels:       %s\n", strings.Join(pairs, "\n              "))
	}

	fmt.Fprintf(w, "Annotations:  <none>\n")
	fmt.Fprintf(w, "API Version:  %s\n", id.APIVersion)
	fmt.Fprintf(w, "Kind:         %s\n", id.Kind)
	fmt.Fprintf(w, "Metadata:\n")
	fmt.Fprintf(w, "  Creation Timestamp:  %s\n", id.CreationTimestamp)
	fmt.Fprintf(w, "  Generation:          %d\n", id.Generation)
	fmt.Fprintf(w, "  Resource Version:    %s\n", id.ResourceVersion)
	fmt.Fprintf(w, "  UID:                 %s\n", id.UID)

	if len(id.Finalizers) == 0 {
		fmt.Fprintf(w, "  Finalizers:          <none>\n")
	} else {
		fmt.Fprintf(w, "  Finalizers:\n")
		for _, f := range id.Finalizers {
			fmt.Fprintf(w, "    %s\n", f)
		}
	}

	fmt.Fprintln(w)
	return nil
}

// printSpecSection prints the Dataset Spec fields (mounts, node affinity).
func printSpecSection(w io.Writer, report *DatasetReport) error {
	spec := report.Spec
	fmt.Fprintf(w, "Spec:\n")

	if len(spec.Mounts) == 0 {
		fmt.Fprintf(w, "  Mounts:  <none>\n")
	} else {
		fmt.Fprintf(w, "  Mounts:\n")
		for _, m := range spec.Mounts {
			fmt.Fprintf(w, "    Mount Point:  %s\n", m.MountPoint)
			if m.Name != "" {
				fmt.Fprintf(w, "    Name:         %s\n", m.Name)
			}
		}
	}

	if len(spec.NodeAffinity.Terms) == 0 {
		fmt.Fprintf(w, "  Node Affinity:  <none>\n")
	} else {
		fmt.Fprintf(w, "  Node Affinity:\n")
		fmt.Fprintf(w, "    Required:\n")
		fmt.Fprintf(w, "      Node Selector Terms:\n")
		for _, term := range spec.NodeAffinity.Terms {
			fmt.Fprintf(w, "        Match Expressions:\n")
			for _, expr := range term {
				fmt.Fprintf(w, "          Key:       %s\n", expr.Key)
				fmt.Fprintf(w, "          Operator:  %s\n", expr.Operator)
				if len(expr.Values) > 0 {
					fmt.Fprintf(w, "          Values:\n")
					for _, v := range expr.Values {
						fmt.Fprintf(w, "            %s\n", v)
					}
				}
			}
		}
	}

	fmt.Fprintln(w)
	return nil
}

// printStatusSection prints the Dataset Status block.
func printStatusSection(w io.Writer, report *DatasetReport) error {
	st := report.Status

	fmt.Fprintf(w, "Status:\n")

	// Cache states
	cs := st.CacheState
	fmt.Fprintf(w, "  Cache States:\n")
	fmt.Fprintf(w, "    Cache Capacity:           %s\n", cs.CacheCapacity)
	fmt.Fprintf(w, "    Cache Hit Ratio:          %s\n", cs.CacheHitRatio)
	fmt.Fprintf(w, "    Cache Throughput Ratio:   %s\n", cs.CacheThroughputRatio)
	fmt.Fprintf(w, "    Cached:                   %s\n", cs.Cached)
	fmt.Fprintf(w, "    Cached Percentage:        %s\n", cs.CachedPercentage)
	fmt.Fprintf(w, "    Local Hit Ratio:          %s\n", cs.LocalHitRatio)
	fmt.Fprintf(w, "    Local Throughput Ratio:   %s\n", cs.LocalThroughputRatio)
	fmt.Fprintf(w, "    Remote Hit Ratio:         %s\n", cs.RemoteHitRatio)
	fmt.Fprintf(w, "    Remote Throughput Ratio:  %s\n", cs.RemoteThroughputRatio)

	// Conditions
	if len(st.Conditions) == 0 {
		fmt.Fprintf(w, "  Conditions:  <none>\n")
	} else {
		fmt.Fprintf(w, "  Conditions:\n")
		for _, c := range st.Conditions {
			fmt.Fprintf(w, "    Last Transition Time:  %s\n", c.LastTransitionTime)
			fmt.Fprintf(w, "    Last Update Time:      %s\n", c.LastUpdateTime)
			fmt.Fprintf(w, "    Message:               %s\n", c.Message)
			fmt.Fprintf(w, "    Reason:                %s\n", c.Reason)
			fmt.Fprintf(w, "    Status:                %s\n", c.Status)
			fmt.Fprintf(w, "    Type:                  %s\n", c.Type)
		}
	}

	fmt.Fprintf(w, "  File Num:                %s\n", st.FileNum)

	// HCFS
	fmt.Fprintf(w, "  Hcfs:\n")
	fmt.Fprintf(w, "    Endpoint:                        %s\n", st.HCFS.Endpoint)
	fmt.Fprintf(w, "    Underlayer File System Version:  %s\n", st.HCFS.UnderlayerFileSystemVersion)

	// Status Mounts (from spec, mirrored in status)
	if len(report.Spec.Mounts) == 0 {
		fmt.Fprintf(w, "  Mounts:  <none>\n")
	} else {
		fmt.Fprintf(w, "  Mounts:\n")
		for _, m := range report.Spec.Mounts {
			fmt.Fprintf(w, "    Mount Point:  %s\n", m.MountPoint)
			if m.Name != "" {
				fmt.Fprintf(w, "    Name:         %s\n", m.Name)
			}
		}
	}

	fmt.Fprintf(w, "  Phase:          %s\n", st.Phase)

	// Runtime summaries
	if len(st.Runtimes) == 0 {
		fmt.Fprintf(w, "  Runtimes:  <none>\n")
	} else {
		fmt.Fprintf(w, "  Runtimes:\n")
		for _, rt := range st.Runtimes {
			fmt.Fprintf(w, "    Category:   %s\n", rt.Category)
			fmt.Fprintf(w, "    Name:       %s\n", rt.Name)
			fmt.Fprintf(w, "    Namespace:  %s\n", rt.Namespace)
			fmt.Fprintf(w, "    Type:       %s\n", rt.Type)
		}
	}

	fmt.Fprintf(w, "  Ufs Total:    %s\n", st.UfsTotal)
	fmt.Fprintf(w, "Events:         <none>\n")

	fmt.Fprintln(w)
	return nil
}

// printResourcesSection prints the per-runtime workload tables and DataOps table.
func printResourcesSection(w io.Writer, report *DatasetReport, wide bool) error {
	if len(report.Runtimes) == 0 {
		fmt.Fprintln(w, "Associated Resources: none")
		fmt.Fprintln(w)
		return nil
	}

	fmt.Fprintln(w, "Associated Resources:")
	fmt.Fprintln(w)

	for _, rt := range report.Runtimes {
		fmt.Fprintf(w, "  Runtime: %s (%s)\n", rt.RuntimeName, rt.RuntimeType)

		if len(rt.Resources) == 0 {
			fmt.Fprintln(w, "    (no resources found)")
			fmt.Fprintln(w)
			continue
		}

		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
		if wide {
			fmt.Fprintln(tw, "    KIND\tNAMESPACE\tNAME\tSTATUS\tAGE\tNODE\tRESTARTS")
		} else {
			fmt.Fprintln(tw, "    KIND\tNAMESPACE\tNAME\tSTATUS\tAGE")
		}
		for _, r := range rt.Resources {
			ns := r.Namespace
			if ns == "" {
				ns = "-"
			}
			if wide {
				node := r.Node
				if node == "" {
					node = "-"
				}
				restarts := r.Restarts
				if restarts == "" {
					restarts = "-"
				}
				fmt.Fprintf(tw, "    %s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					r.Kind, ns, r.Name, r.Status, r.Age, node, restarts)
			} else {
				fmt.Fprintf(tw, "    %s\t%s\t%s\t%s\t%s\n",
					r.Kind, ns, r.Name, r.Status, r.Age)
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}

	if len(report.DataOps) > 0 {
		fmt.Fprintln(w, "  Data Operations:")
		tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
		fmt.Fprintln(tw, "    KIND\tNAMESPACE\tNAME\tSTATUS\tAGE")
		for _, r := range report.DataOps {
			fmt.Fprintf(tw, "    %s\t%s\t%s\t%s\t%s\n",
				r.Kind, r.Namespace, r.Name, r.Status, r.Age)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}

	return nil
}
