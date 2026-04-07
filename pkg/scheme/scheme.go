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

// Package scheme registers the Fluid CRD types and core Kubernetes types
// into a runtime.Scheme that can be used to build a controller-runtime client.
package scheme

import (
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Scheme is the runtime scheme with all types registered.
var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(corev1.AddToScheme(Scheme))
	utilruntime.Must(appsv1.AddToScheme(Scheme))
	utilruntime.Must(fluidv1alpha1.AddToScheme(Scheme))
}
