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
)

var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

func main() {

	apiConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		panic(err)
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		panic(err)
	}

	dClient := dynamic.NewForConfigOrDie(restConfig)

	stopCh := make(chan struct{})
	nodeLister := NewNodeLister(dClient, nil, stopCh)

	time.Sleep(time.Second * 10)

	no, err := nodeLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Println("no", *no[0])

}

func NewNodeLister(dClient *dynamic.DynamicClient, filter func(*apiv1.Node) bool, stopChannel <-chan struct{}) dynamiclister.Lister {
	store, reflector := cache.NewNamespaceKeyedIndexerAndReflector(&cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			return dClient.Resource(nodeGVR).List(context.Background(), options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			return dClient.Resource(nodeGVR).Watch(context.Background(), options)
		},
	}, unstructured.Unstructured{}, time.Second)
	nodeLister := dynamiclister.New(store, nodeGVR)

	go reflector.Run(stopChannel)

	return nodeLister
}
