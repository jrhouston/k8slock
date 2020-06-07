package k8slock

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// number of lockers to run in parallel
var parallelCount = 5

// number of times each locker should lock then unlock
var lockAttempts = 3

func TestLocker(t *testing.T) {
	clientset, err := newKubernetesClientset()
	if err != nil {
		t.Fatalf("could not create kubernetes clientset: %v", err)
	}

	lockers := []sync.Locker{}
	for i := 0; i < parallelCount; i++ {
		locker, err := NewLeaseLocker(clientset, "lock-test")
		if err != nil {
			t.Fatalf("error creating LeaseLocker: %v", err)
		}
		lockers = append(lockers, locker)
	}

	wg := sync.WaitGroup{}
	for _, locker := range lockers {
		wg.Add(1)
		go func(l sync.Locker) {
			defer wg.Done()

			for i := 0; i < lockAttempts; i++ {
				l.Lock()
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				l.Unlock()
			}
		}(locker)
	}
	wg.Wait()
}

func TestLockTTL(t *testing.T) {
	ttlSeconds := 10

	clientset, err := newKubernetesClientset()
	if err != nil {
		t.Fatalf("could not create kubernetes clientset: %v", err)
	}

	locker1, err := NewLeaseLocker(clientset, "ttl-test", TTL(time.Duration(ttlSeconds)*time.Second))
	if err != nil {
		t.Fatalf("error creating LeaseLocker: %v", err)
	}

	locker2, err := NewLeaseLocker(clientset, "ttl-test")
	if err != nil {
		t.Fatalf("error creating LeaseLocker: %v", err)
	}

	locker1.Lock()
	acquired1 := time.Now()
	locker2.Lock()
	acquired2 := time.Now()
	locker2.Unlock()

	diff := acquired2.Sub(acquired1)
	if diff.Seconds() < float64(ttlSeconds) {
		t.Fatal("client was able to acquire lock before the existing one had expired")
	}
}

func newKubernetesClientset() (*kubernetes.Clientset, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = &rest.Config{}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
