package daemon

import (
	"os"
	"syscall"
	"time"

	systemdDaemon "github.com/coreos/go-systemd/daemon"
)

// Ready is called to signal the daemon is ready.
func (c *realComponent) Ready() {
	systemdDaemon.SdNotify(false, "READY=1")
	if os.Getenv("UPSTART_JOB") != "" {
		os.Unsetenv("UPSTART_JOB")
		syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	}
}

// initializeWatchdog will setup the watchdog
func (c *realComponent) initializeWatchdog() {
	c.watchdogLock.Lock()
	defer c.watchdogLock.Unlock()
	interval, err := systemdDaemon.SdWatchdogEnabled(false)
	if err != nil {
		c.r.Warn("unable to query watchdog timer",
			"err", err)
		return
	}
	if interval == 0 {
		// No watchdog configured
		return
	}
	delay := interval / 3
	c.watchdogTicker = time.NewTicker(delay)
	c.TickWatchdog()
	c.r.Info("watchdog enabled", "delay", delay)
}

// stopWatchdog stops the watchdog
func (c *realComponent) stopWatchdog() {
	c.watchdogLock.Lock()
	defer c.watchdogLock.Unlock()
	if c.watchdogTicker != nil {
		c.watchdogTicker.Stop()
		c.watchdogTicker = nil
	}
}

// TickWatchdog will make the watchdog tick.
func (c *realComponent) TickWatchdog() {
	systemdDaemon.SdNotify(false, "WATCHDOG=1")
}

// Watchdog provides a channel for which the watchdog should be ticked
// each time we get a value.
func (c *realComponent) Watchdog() <-chan time.Time {
	c.watchdogLock.Lock()
	defer c.watchdogLock.Unlock()
	if c.watchdogTicker != nil {
		return c.watchdogTicker.C
	}
	return nil
}
