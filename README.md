This is an example for creating a `Lister` using dynamic client for resource of your choice (check code comments for more info). 

## How it works

All nodes:
```bash
$ kubectl get nodes
NAME                            STATUS   ROLES                  AGE   VERSION
k3d-dynamic-corefile-server-0   Ready    control-plane,master   21d   v1.24.4+k3s1

```
Pods in all the namespaces:
```bash
$ kubectl get po -A 
NAMESPACE     NAME                                      READY   STATUS      RESTARTS        AGE
kube-system   helm-install-traefik-crd-vplbc            0/1     Completed   0               21d
kube-system   helm-install-traefik-q26gv                0/1     Completed   1               21d
kube-system   local-path-provisioner-7b7dc8d6f5-q8k6x   1/1     Running     37 (127m ago)   21d
kube-system   coredns-b96499967-tmswr                   1/1     Running     37 (127m ago)   21d
kube-system   traefik-7cd4fcff68-dt82t                  1/1     Running     37 (127m ago)   21d
kube-system   svclb-traefik-98312b71-66crq              2/2     Running     92 (127m ago)   21d
kube-system   metrics-server-668d979685-tbblc           1/1     Running     37 (127m ago)   21d
default       nginx                                     1/1     Running     0               3m39s
```

```bash
$ go run main.go
I0112 11:32:42.385967  105474 shared_informer.go:273] Waiting for caches to sync for generic-nodes-lister
I0112 11:32:42.486633  105474 shared_informer.go:280] Caches are synced for generic-nodes-lister

All nodes:
----------
k3d-dynamic-corefile-server-0

I0112 11:32:42.486738  105474 shared_informer.go:273] Waiting for caches to sync for generic-pods-lister
I0112 11:32:42.587197  105474 shared_informer.go:280] Caches are synced for generic-pods-lister

Pods in `kube-system` namespace:
--------------------------------
coredns-b96499967-tmswr
traefik-7cd4fcff68-dt82t
metrics-server-668d979685-tbblc
helm-install-traefik-crd-vplbc
helm-install-traefik-q26gv
local-path-provisioner-7b7dc8d6f5-q8k6x
svclb-traefik-98312b71-66crq

I0112 11:32:42.587249  105474 shared_informer.go:273] Waiting for caches to sync for generic-pods-lister
I0112 11:32:42.688035  105474 shared_informer.go:280] Caches are synced for generic-pods-lister

Pods in all the namespaces:
---------------------------
traefik-7cd4fcff68-dt82t
metrics-server-668d979685-tbblc
nginx # NOTICE nginx POD HERE FROM THE default NAMESPACE
helm-install-traefik-crd-vplbc
helm-install-traefik-q26gv
local-path-provisioner-7b7dc8d6f5-q8k6x
svclb-traefik-98312b71-66crq
coredns-b96499967-tmswr
```

## Why did I create this?
This is originally a PoC for a [PR I was working on for cluster-autoscaler](https://github.com/kubernetes/autoscaler/pull/5419#discussion_r1071752946). I cleaned the code a bit and added comments to make this a generic PoC/example for anyone who wants to use dynamic listers. 
