package observer

import (
	"fmt"
	"net/http"
	"time"
)

// HTTPProbe is the exported handler object for monitors configured to
// perform HTTP requests
type HTTPProbe struct {
	URL    string
	RCodes []int
}

// Name returns a name that uniquely identifies the monitor
func (p HTTPProbe) Name() string {
	return fmt.Sprintf("%s-%d", p.URL, p.RCodes)
}

// Type returns the type of prober as a string
func (p HTTPProbe) Type() string {
	return "HTTP"
}

// expectedRCode returns true when `got` matches on in
// `HTTPProbe.RCodes`, else returns false
func (p HTTPProbe) expectedRCode(got int) bool {
	for _, c := range p.RCodes {
		if got == c {
			return true
		}
	}
	return false
}

// Do is the request handler for HTTP probes
func (p HTTPProbe) Do(timeout time.Duration) (bool, time.Duration) {
	client := http.Client{Timeout: timeout}
	start := time.Now()
	// TODO(@beautifulentropy): add support for more than HTTP GET
	resp, err := client.Get(p.URL)
	if err != nil {
		return false, time.Since(start)
	}
	// check response code and return
	return p.expectedRCode(resp.StatusCode), time.Since(start)
}
