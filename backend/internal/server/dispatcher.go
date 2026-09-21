package server

import (
	"bytes"
	"net/http"
	"strings"
)

// ModuleDispatcher tries each registered module handler in sequence,
// returning the first non-404 response. This avoids chi v5's restriction
// on multiple Mount("/", ...) calls to the same pattern.
type ModuleDispatcher struct {
	prefix   string
	handlers []http.Handler
}

// NewModuleDispatcher creates a dispatcher that optionally strips a URL prefix
// before forwarding requests to module handlers. Each module's chi.Router
// sees the path without the prefix (e.g., "/api/v1/catalog/products" becomes
// "/catalog/products" when prefix is "/api/v1").
func NewModuleDispatcher(prefix string, handlers ...http.Handler) *ModuleDispatcher {
	return &ModuleDispatcher{prefix: prefix, handlers: handlers}
}

func (d *ModuleDispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, h := range d.handlers {
		var req *http.Request
		if d.prefix != "" {
			req = r.Clone(r.Context())
			req.URL.Path = strings.TrimPrefix(r.URL.Path, d.prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.RequestURI = ""
		} else {
			req = r
		}

		rec := &captureResponse{ResponseWriter: w}
		h.ServeHTTP(rec, req)

		if rec.statusCode != http.StatusNotFound {
			rec.flush()
			return
		}
	}

	http.NotFound(w, r)
}

// captureResponse buffers headers, status, and body so that a failed
// module handler (404) does not write to the real ResponseWriter.
type captureResponse struct {
	http.ResponseWriter
	statusCode int
	header     http.Header
	body       bytes.Buffer
	wrote      bool
}

func (cr *captureResponse) Header() http.Header {
	if cr.header == nil {
		cr.header = make(http.Header)
	}
	return cr.header
}

func (cr *captureResponse) WriteHeader(code int) {
	cr.statusCode = code
	cr.wrote = true
}

func (cr *captureResponse) Write(b []byte) (int, error) {
	if !cr.wrote {
		cr.WriteHeader(http.StatusOK)
	}
	return cr.body.Write(b)
}

func (cr *captureResponse) Unwrap() http.ResponseWriter {
	return cr.ResponseWriter
}

// flush writes the buffered response to the underlying ResponseWriter.
func (cr *captureResponse) flush() {
	if !cr.wrote {
		return
	}
	dst := cr.ResponseWriter.Header()
	for k, v := range cr.header {
		dst[k] = v
	}
	cr.ResponseWriter.WriteHeader(cr.statusCode)
	cr.body.WriteTo(cr.ResponseWriter)
}
