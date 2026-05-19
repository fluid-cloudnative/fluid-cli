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

// gen-docs generates Markdown command reference under docs/reference/.
package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fluid-cloudnative/fluid-cli/cmd/fluid/root"
	"github.com/spf13/cobra/doc"
)

// docHome is used when building the command tree so generated flag defaults
// do not embed a developer machine's home directory.
const docHome = "/home/user"

var kubeCacheDefaultRE = regexp.MustCompile(`\(default "[^"]+/.kube/cache"\)`)

func main() {
	outDir := "docs/reference"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	_ = os.Setenv("HOME", docHome)
	_ = os.Setenv("USERPROFILE", docHome) // Windows-style clients if consulted

	rootCmd := root.NewRootCmd()
	rootCmd.DisableAutoGenTag = true

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("creating output directory: %v", err)
	}

	// Remove stale generated pages so renamed commands do not linger.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		log.Fatalf("reading output directory: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".md" {
			continue
		}
		if err := os.Remove(filepath.Join(outDir, e.Name())); err != nil {
			log.Fatalf("removing stale doc %s: %v", e.Name(), err)
		}
	}

	if err := doc.GenMarkdownTree(rootCmd, outDir); err != nil {
		log.Fatalf("generating docs: %v", err)
	}
	if err := sanitizeGeneratedDocs(outDir); err != nil {
		log.Fatalf("sanitizing docs: %v", err)
	}
	log.Printf("wrote command reference to %s", outDir)
}

func sanitizeGeneratedDocs(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		path := filepath.Join(outDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(data)
		s = kubeCacheDefaultRE.ReplaceAllString(s, `(default "~/.kube/cache")`)
		s = strings.ReplaceAll(s, docHome, "~")
		if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
			return err
		}
	}
	return nil
}
