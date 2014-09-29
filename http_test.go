package httpctx

import (
	"bytes"
	"code.google.com/p/go.net/context"
	"encoding/json"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type mockTransporter struct {
	req   *http.Request
	resp  *http.Response
	err   error
	delay time.Duration
	done  chan interface{}
}

func NewMockTransporter(statusCode int, v interface{}, err error, delay time.Duration) func() transporter {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: statusCode,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	if v != nil {
		data, _ := json.Marshal(v)
		resp.Body = ioutil.NopCloser(bytes.NewReader(data))
	} else {
		data, _ := json.Marshal(map[string]string{"hello": "world"})
		resp.Body = ioutil.NopCloser(bytes.NewReader(data))
	}

	return func() transporter {
		return &mockTransporter{
			resp:  resp,
			err:   err,
			delay: delay,
			done:  make(chan interface{}),
		}
	}
}

func NewMock(resp *http.Response, err error, delay time.Duration) *mockTransporter {
	return &mockTransporter{
		resp:  resp,
		err:   err,
		delay: delay,
		done:  make(chan interface{}),
	}
}

func (m *mockTransporter) CancelRequest(req *http.Request) {
	m.req = req
	if m.done != nil {
		fmt.Println("close(m.done)")
		close(m.done)
		m.done = nil
	}
}

func (m *mockTransporter) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := time.NewTimer(m.delay)
	defer timer.Stop()
	defer m.CancelRequest(req)

	select {
	case <-m.done:
		fmt.Println("<-m.done")
	case <-timer.C:
		fmt.Println("<-timer.C")
	}

	return m.resp, m.err
}

func TestNewRequest(t *testing.T) {
	userAgent := "some-agent"
	var req *http.Request
	var err error

	Convey("When I use #newRequest to create a GET request with params", t, func() {
		params := url.Values{}
		params.Set("hello", "world")
		req, err = newRequest(userAgent, "GET", "http://www.google.com", &params, nil)

		Convey("Then I expect no errors", func() {
			So(err, ShouldBeNil)
		})

		Convey("And I expect the user agent to be set to what I provided", func() {
			So(req.Header.Get("User-Agent"), ShouldEqual, userAgent)
		})

		Convey("And I expect the params to be merged with the url provided", func() {
			So(req.URL.String(), ShouldEqual, "http://www.google.com?hello=world")
		})
	})

	Convey("When I use #newRequest to create a request with an empty path", t, func() {
		_, err = newRequest(userAgent, "GET", "", nil, nil)

		Convey("Then I expect the EmptyPathErr", func() {
			So(err, ShouldEqual, EmptyPathErr)
		})
	})
}

func TestGet(t *testing.T) {
	body := map[string]string{"hello": "world"}

	var err error
	var started int64
	var elapsed int64
	var delay time.Duration
	var result map[string]string

	Convey("Given a responsive site", t, func() {
		err = nil
		delay = 100 * time.Millisecond
		result = map[string]string{}
		newTransporter = NewMockTransporter(200, body, nil, delay)

		Convey("When I call GET", func() {
			started = time.Now().UnixNano()
			err = NewClient().Get(context.Background(), "http://www.google.com", nil, &result)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time >= delay", func() {
				So(elapsed, ShouldBeGreaterThanOrEqualTo, int64(delay))
			})

			Convey("And I expect my response back", func() {
				So(result, ShouldResemble, body)
			})
		})

		Convey("When I prematurely cancel the GET", func() {
			ctx, cancel := context.WithCancel(context.Background())

			started = time.Now().UnixNano()
			go func() { cancel() }()
			err = NewClient().Get(ctx, "http://www.google.com", nil, nil)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time < delay", func() {
				So(elapsed, ShouldBeLessThan, int64(delay))
			})
		})

		Reset(func() {
			newTransporter = makeTransporterFunc
		})
	})

	Convey("Given a site that returns an error code", t, func() {
		err = nil
		delay = 100 * time.Millisecond
		result = map[string]string{}
		newTransporter = NewMockTransporter(500, body, nil, delay)

		Convey("When I call GET", func() {
			started = time.Now().UnixNano()
			err = NewClient().Get(context.Background(), "http://www.google.com", nil, &result)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time >= delay", func() {
				So(elapsed, ShouldBeGreaterThanOrEqualTo, int64(delay))
			})

			Convey("And I expect an error back", func() {
				So(err, ShouldNotBeNil)
				switch v := err.(type) {
				case *ErrorMessage:
					err = v.Unmarshal(&result)
					So(err, ShouldBeNil)
					So(result, ShouldResemble, body)
				default:
					So(true, ShouldBeFalse) // shouldn't happen
				}
			})
		})

		Reset(func() {
			newTransporter = makeTransporterFunc
		})
	})
}

func TestMakeTransporterFunc(t *testing.T) {
	Convey("makeTransporterFunc should return a &http.Transport{}", t, func() {
		So(makeTransporterFunc(), ShouldResemble, &http.Transport{})
	})
}

func TestPost(t *testing.T) {
	body := map[string]string{"hello": "world"}
	var err error
	var started int64
	var elapsed int64
	var delay time.Duration
	var results map[string]string

	Convey("Given a responsive site", t, func() {
		err = nil
		delay = 100 * time.Millisecond
		results = map[string]string{}
		newTransporter = NewMockTransporter(200, nil, nil, delay)

		Convey("When I call POST", func() {
			started = time.Now().UnixNano()
			err = NewClient().Post(context.Background(), "http://www.google.com", body, &results)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time >= delay", func() {
				So(elapsed, ShouldBeGreaterThanOrEqualTo, int64(delay))
			})

			Convey("And I expect the results to match our body", func() {
				So(results, ShouldResemble, body)
			})
		})

		Convey("When I prematurely cancel the POST", func() {
			ctx, cancel := context.WithCancel(context.Background())

			started = time.Now().UnixNano()
			go func() { cancel() }()
			err = NewClient().Post(ctx, "http://www.google.com", nil, nil)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time < delay", func() {
				So(elapsed, ShouldBeLessThan, int64(delay))
			})
		})

		Reset(func() {
			newTransporter = makeTransporterFunc
		})
	})
}

func TestPut(t *testing.T) {
	var resp *http.Response
	var err error
	var started int64
	var elapsed int64
	var delay time.Duration

	Convey("Given a responsive site", t, func() {
		resp = &http.Response{StatusCode: 200}
		err = nil
		delay = 100 * time.Millisecond
		newTransporter = NewMockTransporter(200, nil, nil, delay)

		Convey("When I call POST", func() {
			started = time.Now().UnixNano()
			err = NewClient().Put(context.Background(), "http://www.google.com", nil, nil)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time >= delay", func() {
				So(elapsed, ShouldBeGreaterThanOrEqualTo, int64(delay))
			})
		})

		Convey("When I prematurely cancel the PUT", func() {
			ctx, cancel := context.WithCancel(context.Background())

			started = time.Now().UnixNano()
			go func() { cancel() }()
			err = NewClient().Put(ctx, "http://www.google.com", nil, nil)
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time < delay", func() {
				So(elapsed, ShouldBeLessThan, int64(delay))
			})
		})

		Reset(func() {
			newTransporter = makeTransporterFunc
		})
	})
}

func TestDelete(t *testing.T) {
	var resp *http.Response
	var err error
	var started int64
	var elapsed int64
	var delay time.Duration

	Convey("Given a responsive site", t, func() {
		resp = &http.Response{StatusCode: 200}
		err = nil
		delay = 100 * time.Millisecond
		newTransporter = NewMockTransporter(200, nil, nil, delay)

		Convey("When I call DELETE", func() {
			started = time.Now().UnixNano()
			err = NewClient().Delete(context.Background(), "http://www.google.com")
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time >= delay", func() {
				So(elapsed, ShouldBeGreaterThanOrEqualTo, int64(delay))
			})
		})

		Convey("When I prematurely cancel the PUT", func() {
			ctx, cancel := context.WithCancel(context.Background())

			started = time.Now().UnixNano()
			go func() { cancel() }()
			err = NewClient().Delete(ctx, "http://www.google.com")
			elapsed = time.Now().UnixNano() - started

			Convey("Then I expect response time < delay", func() {
				So(elapsed, ShouldBeLessThan, int64(delay))
			})
		})

		Reset(func() {
			newTransporter = makeTransporterFunc
		})
	})
}
