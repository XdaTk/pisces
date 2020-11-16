package pisces

import (
	"bufio"
	"log"
	"net"
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.Flusher
	Pusher() http.Pusher
}

type responseWriter struct {
	ResponseWriter http.ResponseWriter
	Status         int
	Size           int
	Committed      bool
}

func (r *responseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *responseWriter) Write(bytes []byte) (int, error) {
	if !r.Committed {
		r.WriteHeader(r.Status)
	}

	n, err := r.ResponseWriter.Write(bytes)
	r.Size += n
	return n, err
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if r.Committed {
		log.Printf("response already committed")
		return
	}

	r.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
	r.Committed = true
}

// Hijack implements the http.Hijacker interface to allow an HTTP handler to
// take over the connection.
// See [http.Hijacker](https://golang.org/pkg/net/http/#Hijacker)
func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.ResponseWriter.(http.Hijacker).Hijack()
}

// Flush implements the http.Flusher interface to allow an HTTP handler to flush
// buffered data to the client.
// See [http.Flusher](https://golang.org/pkg/net/http/#Flusher)
func (r *responseWriter) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}

// Pusher returns the http.Pusher that support HTTP/2 server push.
// See [http.Pusher](https://golang.org/pkg/net/http/#Pusher)
func (r *responseWriter) Pusher() http.Pusher {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher
	}

	return nil
}

func (r *responseWriter) reset(w http.ResponseWriter) {
	r.ResponseWriter = w
	r.Size = 0
	r.Status = http.StatusOK
	r.Committed = false
}
