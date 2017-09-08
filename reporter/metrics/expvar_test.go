package metrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"

	"lrg/config"
)

func TestExpvar(t *testing.T) {
	tcpPort := rand.Intn(1000) + 22000
	var configuration Configuration = make([]ExporterConfiguration, 1, 1)
	configuration[0] = &ExpvarConfiguration{
		Listen: config.Addr(fmt.Sprintf("127.0.0.1:%d", tcpPort)),
	}

	m, err := New(configuration, "lrg")
	if err != nil {
		t.Fatalf("New(%v) error:\n%+v", configuration, err)
	}
	m.MustStart()
	defer func() {
		m.Stop()
		if !testing.Short() {
			time.Sleep(1 * time.Second) // Slight race condition...
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", tcpPort))
			if err == nil {
				t.Errorf("Still able to connect to expvar server after stop")
				resp.Body.Close()
			}
		}
	}()

	// Check we can query a value
	c := metrics.NewCounter()
	m.Registry.Register("foo", c)
	c.Inc(47)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", tcpPort))
	if err != nil {
		t.Fatalf("Unable to query expvar server:\n%+v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Unable to read body from expvar server:\n%+v", err)
	}
	var got struct {
		Foo int
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Unable to decode JSON body:\n%s\nError:\n%+v", body, err)
	}
	if got.Foo != 47 {
		t.Fatalf("Expected Foo == 47 but got %d instead", got.Foo)
	}
}
