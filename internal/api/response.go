package api

import "net/http"

// ResponseRecorder wraps an http.ResponseWriter to capture the status code and
// bytes written. It is used by middleware that needs to observe responses
// without altering their content.
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Bytes      int
}

// WriteHeader captures the first status code written and forwards to the
// underlying ResponseWriter.
func (r *ResponseRecorder) WriteHeader(statusCode int) {
	if r.StatusCode != 0 {
		return
	}

	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write captures bytes written and defaults the status code to 200 OK if
// WriteHeader was never called.
func (r *ResponseRecorder) Write(data []byte) (int, error) {
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}

	n, err := r.ResponseWriter.Write(data)
	r.Bytes += n
	return n, err
}
