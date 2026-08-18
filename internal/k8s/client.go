// Copyright Meshery Authors
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

// Package k8s provides a minimal client-go based source of live cluster
// topology. It is used as a fallback when Meshery's MeshSync has not synced a
// cluster, so the topology resource still reflects real running state.
package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/AkashNaickar/meshery-mcp-demo/internal/meshery"
)

// Options configures a Client.
type Options struct {
	// KubeconfigPath is the path to the kubeconfig file.
	KubeconfigPath string
	// Context is the kubeconfig context to use; empty uses the current context.
	Context string
	// Namespace is the namespace to read topology from; empty uses "default".
	Namespace string
}

// Client reads live cluster topology using the standard Kubernetes API.
type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

// New builds a Client from a kubeconfig file. It returns an error when the
// kubeconfig cannot be loaded or no usable context is found.
func New(opts Options) (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.KubeconfigPath != "" {
		loadingRules.ExplicitPath = opts.KubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}

	namespace := opts.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return &Client{clientset: clientset, namespace: namespace}, nil
}

// ListTopology returns the live resources in the configured namespace as
// topology components: Deployments, ReplicaSets, Pods, and Services. It
// satisfies meshery.TopologySource.
func (c *Client) ListTopology(ctx context.Context) ([]meshery.TopologyComponent, error) {
	var out []meshery.TopologyComponent

	deployments, err := c.clientset.AppsV1().Deployments(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	for _, d := range deployments.Items {
		out = append(out, meshery.TopologyComponent{ID: string(d.UID), Kind: "Deployment", Name: d.Name})
	}

	replicaSets, err := c.clientset.AppsV1().ReplicaSets(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list replica sets: %w", err)
	}
	for _, rs := range replicaSets.Items {
		out = append(out, meshery.TopologyComponent{ID: string(rs.UID), Kind: "ReplicaSet", Name: rs.Name})
	}

	pods, err := c.clientset.CoreV1().Pods(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	for _, p := range pods.Items {
		out = append(out, meshery.TopologyComponent{ID: string(p.UID), Kind: "Pod", Name: p.Name})
	}

	services, err := c.clientset.CoreV1().Services(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	for _, s := range services.Items {
		out = append(out, meshery.TopologyComponent{ID: string(s.UID), Kind: "Service", Name: s.Name})
	}

	return out, nil
}
