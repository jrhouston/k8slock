package k8slock

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/pointer"
)

// number of lockers to run in parallel
var parallelCount = 5

// number of times each locker should lock then unlock
var lockAttempts = 3

var clientset = fake.NewSimpleClientset()

func TestLocker(t *testing.T) {
	lockers := []sync.Locker{}
	for i := 0; i < parallelCount; i++ {
		locker, err := NewLocker("lock-test", Clientset(clientset))
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

	locker1, err := NewLocker("ttl-test", TTL(time.Duration(ttlSeconds)*time.Second), Clientset(clientset))
	if err != nil {
		t.Fatalf("error creating LeaseLocker: %v", err)
	}

	locker2, err := NewLocker("ttl-test", Clientset(clientset))
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

func TestPanicErrorWrap(t *testing.T) {
	locker, err := NewLocker("wrap-test")
	if err != nil {
		t.Fatalf("error creating LeaseLocker: %v", err)
	}

	_ = locker.leaseClient.Delete(context.Background(), locker.name, metav1.DeleteOptions{})

	var panicErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = r.(error)
			}
		}()
		locker.Unlock()
	}()

	if panicErr == nil {
		t.Fatalf("expected panic, but got none")
	}

	checkErr := new(k8serrors.StatusError)
	if !errors.As(panicErr, &checkErr) {
		t.Fatalf("expected StatusError, but got: %v", panicErr)
	}
}

func TestSkipLeaseCreation(t *testing.T) {
	leaseName := "skip-create"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name: leaseName,
		},
		Spec: coordinationv1.LeaseSpec{
			LeaseTransitions: pointer.Int32(0),
		},
	}
	_, err := clientset.CoordinationV1().Leases("default").Create(context.Background(), lease, metav1.CreateOptions{})

	locker, err := NewLocker(leaseName, Clientset(clientset), CreateLease(false))
	if err != nil {
		t.Fatalf("error creating LeaseLocker: %v", err)
	}

	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("panic when locking: %v", err)
		}
	}()
	locker.Lock()
	locker.Unlock()
}
