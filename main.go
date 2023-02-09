package main

import (
	"context"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamiclister "k8s.io/client-go/dynamic/dynamiclister"
	kube_client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

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

	kubeClient := createKubeClient(restConfig)
	dClient := dynamic.New(kubeClient.Discovery().RESTClient())

	// as an alternative to dynamic.New(), you can uncomment
	//  the following line to create a dynamic client
	// dClient := dynamic.NewForConfigOrDie(restConfig)

	// For stopping the reflector
	stopCh := make(chan struct{})

	// Note that for `core` group we use ""
	// Resource should be in plural form e.g., pods, deployments etc.,
	// Ref: https://github.com/kubernetes/client-go/issues/737
	var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

	nodeLister := NewDynamicLister(dClient, stopCh, nodeGVR, apiv1.NamespaceAll)

	// Get all the nodes
	nos, err := nodeLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Println("") // add some space after the last line for better display
	fmt.Println("All nodes:")
	fmt.Println("----------")
	for _, n := range nos {
		fmt.Println(n.GetName())
	}
	fmt.Println("") // add some space after the last line for better display

	// Note that for `core` group we use ""
	// Resource should be in plural form e.g., pods, deployments etc.,
	// Ref: https://github.com/kubernetes/client-go/issues/737
	var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

	// Limit lister to a particular namespace (use only for namespaced resources)
	podLister := NewDynamicLister(dClient, stopCh, podGVR, "kube-system")

	// Get all the pods
	po, err := podLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Println("") // add some space after the last line for better display
	fmt.Println("Pods in `kube-system` namespace:")
	fmt.Println("--------------------------------")
	for _, p := range po {
		fmt.Println(p.GetName())
	}
	fmt.Println("") // add some space after the last line for better display

	// Limit lister to a particular namespace (use only for namespaced resources)
	allPodsLister := NewDynamicLister(dClient, stopCh, podGVR, apiv1.NamespaceAll)

	// Get all the pods in all the namespaces
	allPo, err := allPodsLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	fmt.Println("") // add some space after the last line for better display
	fmt.Println("Pods in all the namespaces:")
	fmt.Println("---------------------------")
	for _, p := range allPo {
		fmt.Println(p.GetName())
	}
	fmt.Println("") // add some space after the last line for better display

	// Uncomment to use the CRD Lister
	// // CRD Lister
	// crdLister := NewDynamicCRDLister(dClient, stopCh)

	// // Get CRDs by specifying the key in the format `<group>/Kind` (<- Kind needs to be in CamelCase)
	// // Note that this is quite different from specifying the key as `<namespace>/<name>`
	// no, err := crdLister.Get("traefik.containo.us/ServersTransport")
	// if err != nil {
	// 	panic(err)
	// }
	// // pretty print
	// output, err := json.MarshalIndent(no, "", "  ")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("CRD", string(output))

}

func NewDynamicLister(dClient *dynamic.DynamicClient, stopChannel <-chan struct{}, gvr schema.GroupVersionResource, namespace string) dynamiclister.Lister {

	var lister func(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
	var watcher func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	if namespace == apiv1.NamespaceAll {
		lister = dClient.Resource(gvr).List
		watcher = dClient.Resource(gvr).Watch
	} else {
		// For lister limited to a particular namespace
		lister = dClient.Resource(gvr).Namespace(namespace).List
		watcher = dClient.Resource(gvr).Namespace(namespace).Watch
	}

	// NewNamespaceKeyedIndexerAndReflector can be
	// used for both namespace and cluster scoped resources
	store, reflector := cache.NewNamespaceKeyedIndexerAndReflector(&cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			return lister(context.Background(), options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			return watcher(context.Background(), options)
		},
	}, unstructured.Unstructured{}, time.Hour)
	nodeLister := dynamiclister.New(store, gvr)

	// Run reflector in the background so that we get new updates from the api-server
	go reflector.Run(stopChannel)

	// Wait for reflector to sync the cache for the first time
	// TODO: check if there's a better way to do this (listing all the nodes seems wasteful)
	// Note: Based on the docs WaitForNamedCacheSync seems to be used to check if an informer has synced
	// but the function is generic enough so we can use
	// it for reflectors as well
	synced := cache.WaitForNamedCacheSync(fmt.Sprintf("generic-%s-lister", gvr.Resource), stopChannel, func() bool {
		no, err := nodeLister.List(labels.Everything())
		if err != nil {
			klog.Error("err", err)
		}
		return len(no) > 0
	})
	if !synced {
		klog.Error("couldn't sync cache")
	}

	return nodeLister
}

func NewDynamicCRDLister(dClient *dynamic.DynamicClient, stopChannel <-chan struct{}) dynamiclister.Lister {

	var lister func(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error)
	var watcher func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}

	lister = dClient.Resource(gvr).List
	watcher = dClient.Resource(gvr).Watch
	store := cache.NewIndexer( /* Key Func*/ func(obj interface{}) (string, error) {
		uo := obj.(*unstructured.Unstructured)
		o := uo.Object
		group, found, err := unstructured.NestedString(o, "spec", "group")
		if !found {
			fmt.Printf("didn't find value on %v", uo.GetName())
		}
		if err != nil {
			fmt.Printf("err: %v", err)
		}

		names, found, err := unstructured.NestedStringMap(o, "spec", "names")
		if !found {
			fmt.Printf("didn't find value on %v", uo.GetName())
		}
		if err != nil {
			fmt.Printf("err: %v", err)
		}

		// Key is <group>/<Kind> as opposed to <namespace>/name
		// This is so that you can find CRD just using Kind and API Group
		// instead of knowing the name
		return group + "/" + names["kind"], nil
	}, cache.Indexers{"group": /* Index Func */ func(obj interface{}) ([]string, error) {
		uo := obj.(*unstructured.Unstructured)
		o := uo.Object
		group, found, err := unstructured.NestedString(o, "spec", "group")
		if !found {
			fmt.Printf("didn't find value on %v", uo.GetName())
		}
		if err != nil {
			return []string{""}, fmt.Errorf("err: %v", err)
		}
		/* Index by APi Group of the CRD */
		return []string{group}, nil
	}})

	lw := &cache.ListWatch{
		ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
			return lister(context.Background(), options)
		},
		WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
			return watcher(context.Background(), options)
		},
	}

	reflector := cache.NewReflector(lw, unstructured.Unstructured{}, store, time.Hour)

	crdLister := dynamiclister.New(store, gvr)

	// Run reflector in the background so that we get new updates from the api-server
	go reflector.Run(stopChannel)

	// Wait for reflector to sync the cache for the first time
	// TODO: check if there's a better way to do this (listing all the nodes seems wasteful)
	// Note: Based on the docs WaitForNamedCacheSync seems to be used to check if an informer has synced
	// but the function is generic enough so we can use
	// it for reflectors as well
	synced := cache.WaitForNamedCacheSync(fmt.Sprintf("generic-%s-lister", gvr.Resource), stopChannel, func() bool {
		no, err := crdLister.List(labels.Everything())
		if err != nil {
			klog.Error("err", err)
		}
		return len(no) > 0
	})
	if !synced {
		klog.Error("couldn't sync cache")
	}

	return crdLister
}

// createKubeClient mimics function of the same name in cluster-autoscaler
func createKubeClient(kubeConfig *rest.Config) kube_client.Interface {
	return kube_client.NewForConfigOrDie(kubeConfig)
}
