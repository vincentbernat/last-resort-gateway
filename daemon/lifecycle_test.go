package daemon

import (
	"testing"

	"lrg/reporter"
)

func TestTerminate(t *testing.T) {
	r := reporter.NewMock()
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}
	c.Start()

	select {
	case <-c.Terminated():
		t.Fatalf("Terminated() was closed while we didn't request termination")
	default:
		// OK
	}

	c.Terminate()
	select {
	case _, ok := <-c.Terminated():
		if ok {
			t.Fatalf("Terminated() returned an unexpected value")
		}
		// OK
	default:
		t.Fatalf("Terminated() wasn't closed while we requested it to be")
	}

	c.Terminate() // Can be called several times.
}

func TestStop(t *testing.T) {
	r := reporter.NewMock()
	c, err := New(r)
	if err != nil {
		t.Fatalf("New() error:\n%+v", err)
	}
	c.Start()

	select {
	case <-c.Terminated():
		t.Fatalf("Terminated() was closed while we didn't request termination")
	default:
		// OK
	}

	c.Stop()
	select {
	case _, ok := <-c.Terminated():
		if ok {
			t.Fatalf("Terminated() returned an unexpected value")
		}
		// OK
	default:
		t.Fatalf("Terminated() wasn't closed while we requested it to be")
	}
}
