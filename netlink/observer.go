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

// observerSubComponent send notifications to a registered subscriber.
type observerSubComponent struct {
	callback     func(Notification)
	callbackLock sync.RWMutex
	subscribed   chan struct{}
	once         sync.Once
}

// newObserver returns a new observer subcomponent.
func newObserver() observerSubComponent {
	return observerSubComponent{subscribed: make(chan struct{})}
}

// Subscribe or replace a callback. When replacing, old notifications are not sent.
func (c *observerSubComponent) Subscribe(cb func(Notification)) {
	c.callbackLock.Lock()
	defer c.callbackLock.Unlock()
	c.callback = cb
	c.once.Do(func() {
		close(c.subscribed)
	})
}

// notify will notify the eventual subscriber of a route update.
func (c *observerSubComponent) notify(notification Notification) {
	c.callbackLock.RLock()
	defer c.callbackLock.RUnlock()
	if c.callback != nil {
		c.callback(notification)
	}
}
