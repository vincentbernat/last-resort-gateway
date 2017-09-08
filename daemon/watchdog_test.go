// +build !race

package daemon

import (
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"lrg/reporter"
)

func TestWatchdog(t *testing.T) {
	testDir, err := ioutil.TempDir("", "systemd")
	if err != nil {
		t.Fatalf("TempDir() error:\n%+v", err)
	}
	defer os.RemoveAll(testDir)
	notifySocket := path.Join(testDir, "notify-socket.sock")
	laddr := net.UnixAddr{
		Name: notifySocket,
		Net:  "unixgram",
	}
	conn, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		t.Fatalf("ListenUnixgramm() error:\n%+v", err)
	}
	defer conn.Close()
	if err := os.Setenv("NOTIFY_SOCKET", notifySocket); err != nil {
		t.Fatalf("SetEnv() error:\n%+v", err)
	}
	if err := os.Setenv("WATCHDOG_USEC", "20000"); err != nil { // 20 ms
		t.Fatalf("SetEnv() error:\n%+v", err)
	}
	r := reporter.NewMock()

	var countWatchdog, countReady int
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}
	stopped1 := make(chan bool)
	stopped2 := make(chan bool)
	go func() {
	outer:
		for {
			select {
			case <-c.Terminated():
				close(stopped1)
				break outer
			default:
			}

			var buf [1024]byte
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buf[:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			if err != nil {
				continue
			}
			switch string(buf[:n]) {
			case "READY=1":
				countReady++
			case "WATCHDOG=1":
				countWatchdog++
			default:
				t.Errorf("Unix socket received unexpected %q", string(buf[:n]))
			}
		}
	}()

	go func() {
	outer:
		for {
			select {
			case <-c.Terminated():
				close(stopped2)
				break outer
			case <-c.Watchdog():
				c.TickWatchdog()
			}
		}
	}()

	c.Start()
	c.Ready()
	time.Sleep(500 * time.Millisecond)
	c.Stop()
	<-stopped1
	<-stopped2
	// We should get 75 ticks
	if countWatchdog < 5 || countWatchdog > 100 {
		t.Errorf("%d WATCHDOG=1 received, expected 5 < x < 100", countWatchdog)
	}
	if countReady != 1 {
		t.Errorf("%d READY=1 received, expected exactly 1", countReady)
	}
}
