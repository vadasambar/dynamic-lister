package main

import (
	"context"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamiclister "k8s.io/client-go/dynamic/dynamiclister"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// Note that for `core` group we use ""
// Resource should be in plural form e.g., pods, deployments etc.,
// Ref: https://github.com/kubernetes/client-go/issues/737
var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

func main() {

	// Check and load kubeconfig from the path set
	// in KUBECONFIG env variable (if not use default path of ~/.kube/config)
	apiConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		panic(err)
	}

	// Create rest config from kubeconfig
	restConfig, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		panic(err)
	}

	dClient := dynamic.NewForConfigOrDie(restConfig)

	// For stopping the reflector
	stopCh := make(chan struct{})

	nodeLister := NewNodeLister(dClient, nil, stopCh)

	// Get all the nodes
	no, err := nodeLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Println("node", no[0].GetName())
}

func NewNodeLister(dClient *dynamic.DynamicClient, filter func(*apiv1.Node) bool, stopChannel <-chan struct{}) dynamiclister.Lister {
	// NewNamespaceKeyedIndexerAndReflector can be
	// used for both namespace and cluster scoped resources
	store, reflector := cache.NewNamespaceKeyedIndexerAndReflector(&cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			return dClient.Resource(nodeGVR).List(context.Background(), options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			return dClient.Resource(nodeGVR).Watch(context.Background(), options)
		},
	}, unstructured.Unstructured{}, time.Hour)
	nodeLister := dynamiclister.New(store, nodeGVR)

	// Run reflector in the background so that we get new updates from the api-server
	go reflector.Run(stopChannel)

	// Wait for reflector to sync the cache for the first time
	// TODO: check if there's a better way to do this (listing all the nodes seems wasteful)
	// Note: Based on the docs WaitForNamedCacheSync seems to be used to check if an informer has synced
	// but the function is generic enough so we can use
	// it for reflectors as well
	cache.WaitForNamedCacheSync("node-lister", stopChannel, func() bool {
		no, err := nodeLister.List(labels.Everything())
		if err != nil {
			klog.Error("err", err)
		}
		return len(no) > 0
	})

	return nodeLister
}
