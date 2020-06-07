# k8slock

k8s lock is a Go module that enables distributed locking by implementing the [sync.Locker](https://golang.org/pkg/sync/#Locker) interface using the the [Lease](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#lease-v1-coordination-k8s-io) resource from the Kubernetes coordination API. 

If you want to use Kubernetes to create a simple distributed lock, this module is for you.