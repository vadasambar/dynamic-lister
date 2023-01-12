This is an example for creating a `Lister` using dynamic client for resource of your choice. 

## How it works

```
$ kubectl get nodes
NAME                            STATUS   ROLES                  AGE   VERSION
k3d-dynamic-corefile-server-0   Ready    control-plane,master   21d   v1.24.4+k3s1

```

```
$ go run main.go
I0112 10:43:26.123920   77494 shared_informer.go:273] Waiting for caches to sync for node-lister
I0112 10:43:26.224850   77494 shared_informer.go:280] Caches are synced for node-lister
node k3d-dynamic-corefile-server-0
```