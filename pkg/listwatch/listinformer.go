// Copyright 2019 Istio Authors
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

// Provides multiple namespace listerWatcher. This implementation is Largely from
// https://github.com/coreos/prometheus-operator/pkg/listwatch/listwatch.go

package listwatch

import (
	"sync"
	"time"

	"istio.io/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

/*
 * This type adapts the events from the informer into events which would come
 * from a Watch, in this way we can use this as a lower level cache handling
 * individual namespaces to feed a higher level cache aggregating across all
 * the namespaces.
 *
 * This type will also support draining in that once the namespace has been removed
 * from the aggregated cache it will issue DELETED events for each entry within this
 * namespace so that the upper level cache is consistent.
 */
type listerInformer struct {
	namespace        string
	informer         cache.SharedInformer
	lllw             *listLenListerWatcher
	events           func(<-chan struct{}, *watch.Event)
	stoppedInformer  chan struct{}
	stopInformerOnce sync.Once
	listerHasStopped chan struct{}
	stopListerOnce   sync.Once
	draining         bool
	drained          func(string)
	lock             sync.RWMutex
}

func newListerInformer(namespace string, f func(string) cache.ListerWatcher, exampleObject runtime.Object,
	resyncPeriod time.Duration, events func(<-chan struct{}, *watch.Event), drained func(string)) *listerInformer {

	lllw := newListLenListerWatcher(f(namespace))
	informer := cache.NewSharedInformer(lllw, exampleObject, resyncPeriod)

	li := &listerInformer{
		namespace:        namespace,
		informer:         informer,
		lllw:             lllw,
		events:           events,
		stoppedInformer:  make(chan struct{}),
		listerHasStopped: make(chan struct{}),
		drained:          drained,
	}
	informer.AddEventHandler(li)
	go informer.Run(li.stoppedInformer)
	return li
}

// The lister informer is considered synced if the underlying
// informer has synced *and* we have passed sufficient ADDED events
// to the higher level cache to match the underlying List count.
func (li *listerInformer) hasSynced() bool {
	return li.informer.HasSynced() && li.lllw.hasReachedListCount()
}

func (li *listerInformer) isDraining() bool {
	li.lock.RLock()
	defer li.lock.RUnlock()
	return li.draining
}

func (li *listerInformer) isStopped() bool {
	select {
	case <-li.listerHasStopped:
		return true
	default:
	}
	return false
}

func (li *listerInformer) drain() {
	// If we are already draining then there is nothing to do
	shouldDrain := func() bool {
		li.lock.Lock()
		defer li.lock.Unlock()
		if li.draining {
			return false
		}
		li.draining = true
		// We are draining.  Stop the underlying informer so it
		// cannot interfere with our Delete events
		li.stopInformer()
		return true
	}()
	if shouldDrain {
		go func() {
			defer li.drained(li.namespace)
			defer li.stopListerInformer()

			// Issue Delete events for each entry remaining in the store
			store := li.informer.GetStore()
			resourcesToDrain := store.List()
			for _, resource := range resourcesToDrain {
				li.OnDelete(resource)
			}
		}()
	}
}

func (li *listerInformer) stopInformer() {
	close(li.stoppedInformer)
}

func (li *listerInformer) stopListerInformer() {
	li.stopListerOnce.Do(func() {
		close(li.listerHasStopped)
	})
}

func (li *listerInformer) stop() {
	li.stopInformerOnce.Do(func() {
		li.stopListerInformer()
		li.stopInformer()
	})
}

func (li *listerInformer) newWatchEvent(eventType watch.EventType, obj interface{}) (*watch.Event, bool) {
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if ok {
		obj = tombstone.Obj
	}
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		log.Warnf("Unexpected event type from cache %T, ignoring", obj)
		return nil, false
	}
	return &watch.Event{
		eventType, runtimeObj,
	}, true
}

func (li *listerInformer) sendEvent(event *watch.Event) {
	li.events(li.listerHasStopped, event)
}

// Adapt an OnAdd event into a watch ADDED event
func (li *listerInformer) OnAdd(obj interface{}) {
	if watchEvent, ok := li.newWatchEvent(watch.Added, obj); ok {
		li.sendEvent(watchEvent)
		li.lllw.incAddCount()
	}
}

// Adapt an OnUpdate event into a watch MODIFIED event
func (li *listerInformer) OnUpdate(oldObj, newObj interface{}) {
	if watchEvent, ok := li.newWatchEvent(watch.Modified, newObj); ok {
		li.sendEvent(watchEvent)
	}
}

// Adapt an OnDelete event into a watch DELETED event
func (li *listerInformer) OnDelete(obj interface{}) {

	if watchEvent, ok := li.newWatchEvent(watch.Deleted, obj); ok {
		li.sendEvent(watchEvent)
	}
}
