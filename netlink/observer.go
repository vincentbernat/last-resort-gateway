package netlink

import (
	"sync"

	"github.com/vishvananda/netlink"
)

// observerSubComponent manages a list of subscribers and allow to
// subscribe/unsubscribe to them.
type observerSubComponent struct {
	callbacks    map[string]func(netlink.RouteUpdate)
	callbackLock sync.RWMutex
}

// newObserver returns a new observer subcomponent.
func newObserver() observerSubComponent {
	return observerSubComponent{
		callbacks: make(map[string]func(netlink.RouteUpdate)),
	}
}

// Subscribe adds or replace a callback to the callback chain. The
// callback is identified by the provided string.
func (c *observerSubComponent) Subscribe(id string, cb func(netlink.RouteUpdate)) {
	c.callbackLock.Lock()
	defer c.callbackLock.Unlock()
	c.callbacks[id] = cb
}

// Unsubscribe remove a callback from the callback chain. It does
// nothing if the callback doesn't exist.
func (c *observerSubComponent) Unsubscribe(id string) {
	c.callbackLock.Lock()
	defer c.callbackLock.Unlock()
	delete(c.callbacks, id)
}

// notify will notify all subscribers of a route update. It returns
// the number of updates.
func (c *observerSubComponent) notify(update netlink.RouteUpdate) int {
	count := 0
	c.callbackLock.RLock()
	defer c.callbackLock.RUnlock()
	for _, cb := range c.callbacks {
		count++
		cb(update)
	}
	return count
}
