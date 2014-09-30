package httpctx

import (
	"bytes"
	"code.google.com/p/go.net/context"
	"encoding/json"
	"errors"
	. "github.com/savaki/go-debug"
	"io"
	"net/http"
	"net/url"
)

var debug = Debug("httpctx")

// handles creation of http.Transport instances; provides simple hook that can be overridden for testing
var newTransporter func() transporter = makeTransporterFunc

var (
	EmptyPathErr = errors.New("unable to construct a request from an empty url")
)

func makeTransporterFunc() transporter {
	return &http.Transport{}
}

type HttpClient interface {
	Get(ctx context.Context, path string, params *url.Values, v interface{}) error
	Post(ctx context.Context, path string, body interface{}, v interface{}) error
	Put(ctx context.Context, path string, body interface{}, v interface{}) error
	Delete(ctx context.Context, path string) error
}

// creates a new HttpClient without authentication
func NewClient() HttpClient {
	return WithAuthFunc(nil)
}

// creates a new HttpClient that uses the specified algorithm for authentication
func WithAuthFunc(authFunc AuthFunc) HttpClient {
	return &client{
		authFunc:  authFunc,
		UserAgent: "httpctx-go:0.1",
	}
}

type client struct {
	authFunc  AuthFunc
	UserAgent string
}

// path - a fully qualified http(s) path
// params - an optional pointer to url.Values
// v - an optional struct instance that we will unmarshal to e.g. json.NewDecoder(...).Decode(v)
func (h *client) Get(ctx context.Context, path string, params *url.Values, v interface{}) error {
	return h.Do(ctx, "GET", path, params, nil, v)
}

// path - a fully qualified http(s) path
// params - an optional pointer to url.Values
// data - an optional struct to pass in as json encoded data
// v - an optional struct instance that we will unmarshal to e.g. json.NewDecoder(...).Decode(v)
func (h *client) Post(ctx context.Context, path string, data interface{}, v interface{}) error {
	return h.Do(ctx, "POST", path, nil, data, v)
}

// path - a fully qualified http(s) path
// params - an optional pointer to url.Values
// data - an optional struct to pass in as json encoded data
// v - an optional struct instance that we will unmarshal to e.g. json.NewDecoder(...).Decode(v)
func (h *client) Put(ctx context.Context, path string, data interface{}, v interface{}) error {
	return h.Do(ctx, "PUT", path, nil, data, v)
}

// path - a fully qualified http(s) path
func (h *client) Delete(ctx context.Context, path string) error {
	return h.Do(ctx, "DELETE", path, nil, nil, nil)
}

type response struct {
	resp *http.Response
	err  error
}

func newRequest(userAgent, method, path string, params *url.Values, payload interface{}) (*http.Request, error) {
	if path == "" {
		return nil, EmptyPathErr
	}

	// marshal body if data != nil
	body, err := toJson(payload)
	if err != nil {
		return nil, err
	}

	// update path with params if params != nil
	_path := path
	if params != nil {
		uri, err := url.Parse(path)
		if err != nil {
			return nil, err
		}

		uri.RawQuery = params.Encode()
		_path = uri.String()
	}

	req, _ := http.NewRequest(method, _path, body)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (h *client) handle(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	// send the request on a new custom transport; result will be pumped to the ch channel
	tr := newTransporter()
	ch := make(chan response, 1)
	defer close(ch)

	debug("%s %s", req.Method, req.URL.String())

	go func() {
		resp, err := tr.RoundTrip(req)
		ch <- response{resp: resp, err: err}
	}()

	select {
	case <-ctx.Done():
		tr.CancelRequest(req)
		tr.CloseIdleConnections()
		<-ch
		err = ctx.Err()
		return
	case r := <-ch:
		resp = r.resp
		err = r.err
	}
	return
}

func (h *client) Do(ctx context.Context, method, path string, params *url.Values, payload interface{}, v interface{}) error {
	// 1. create a new request
	req, err := newRequest(h.UserAgent, method, path, params, payload)
	if err != nil {
		return err
	}

	// 2. perform whatever authorization may be required
	if h.authFunc != nil {
		req = h.authFunc(req)
	}

	// 3. execute the request
	resp, err := h.handle(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 4. manually follow a 302 redirect
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		return h.Get(ctx, location, nil, v)
	}

	// 5. process the results
	if data := toByteArray(resp.Body); !ok(resp.StatusCode) {
		err = &ErrorMessage{StatusCode: resp.StatusCode, Data: data}

	} else if v != nil {
		err = json.Unmarshal(data, v)
	}

	return err
}

func ok(statusCode int) bool {
	firstDigit := statusCode / 100
	return firstDigit == 2
}

func toByteArray(body io.Reader) []byte {
	buf := bytes.NewBuffer([]byte{})
	io.Copy(buf, body)
	return buf.Bytes()
}
