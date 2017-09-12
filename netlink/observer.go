package netlink

import (
	"sync"

	"github.com/vishvananda/netlink"
)

// Notification represents a notification to be sent to a
// subscriber. Only one of each member is set at a time: either the
// notification contains a route update, or it is the start of a new
// RIB or the end of the initial RIB.
type Notification struct {
	RouteUpdate *netlink.RouteUpdate // Route update or nil if no route
	StartOfRIB  bool                 // Previous RIB should be discarded
	EndOfRIB    bool                 // End of initial RIB
}

// observerSubComponent manages a list of subscribers and allow to
// subscribe/unsubscribe to them.
type observerSubComponent struct {
	callbacks    map[string]func(Notification)
	callbackLock sync.RWMutex
}

// newObserver returns a new observer subcomponent.
func newObserver() observerSubComponent {
	return observerSubComponent{
		callbacks: make(map[string]func(Notification)),
	}
}

// Subscribe adds or replace a callback to the callback chain. The
// callback is identified by the provided string.
func (c *observerSubComponent) Subscribe(id string, cb func(Notification)) {
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
func (c *observerSubComponent) notify(notification Notification) int {
	count := 0
	c.callbackLock.RLock()
	defer c.callbackLock.RUnlock()
	for _, cb := range c.callbacks {
		count++
		cb(notification)
	}
	return count
}
