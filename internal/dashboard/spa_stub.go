//go:build !spa

package dashboard

import "net/http"

// spaFS is nil when built without -tags spa.
var spaFS interface{}

func spaAvailable() bool { return false }

// SPAHandler returns a handler that reports the SPA is not available.
func SPAHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "SPA not available (build without -tags spa)", http.StatusNotFound)
	})
}
