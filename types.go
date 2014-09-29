package httpctx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type transporter interface {
	RoundTrip(*http.Request) (*http.Response, error)
	CancelRequest(*http.Request)
}

// authorizes a given request; generally you should return the request you are provided, but in cases where you must,
// you can also return a new *http.Request instance
type AuthFunc func(*http.Request) *http.Request

// captures the status code and payload contents for non-200 status messages
type ErrorMessage struct {
	StatusCode int
	Data       []byte
}

// used like json call of same name
func (e ErrorMessage) Unmarshal(v interface{}) error {
	return json.NewDecoder(bytes.NewReader(e.Data)).Decode(v)
}

// the status code wrapped into a string
func (e ErrorMessage) Error() string {
	return fmt.Sprintf("returned status code => %d", e.StatusCode)
}
