package executors

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gentleman.v2"
)

func TestRetryRequest(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("foo", r.Header.Get("foo"))
		_, _ = fmt.Fprintln(w, "Hello, world")
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.SetHeader("foo", "bar")
	req.URL(ts.URL)
	req.Use(NewRetry(nil, nil))

	res, err := req.Send()
	require.NoError(t, err)
	require.True(t, res.Ok)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "bar", res.Header.Get("foo"))
	require.Equal(t, 3, calls)
}

func TestRetryRequestWithPayload(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		buf, _ := ioutil.ReadAll(r.Body)
		_, _ = fmt.Fprintln(w, string(buf))
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Method("POST")
	req.BodyString("Hello, world")
	req.Use(NewRetry(nil, nil))

	res, err := req.Send()
	require.NoError(t, err)
	require.True(t, res.Ok)
	require.Equal(t, int64(13), res.RawResponse.ContentLength)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "Hello, world\n", res.String())
	require.Equal(t, 3, calls)
}

func TestRetryServerError(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Use(NewRetry(nil, nil))

	res, err := req.Send()
	require.NoError(t, err)
	require.False(t, res.Ok)
	require.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	require.Equal(t, 4, calls)
}

func TestRetryNetworkError(t *testing.T) {
	req := gentleman.NewRequest()
	req.URL("http://127.0.0.1:1")
	req.Use(NewRetry(nil, nil))

	res, err := req.Send()
	require.Error(t, err)
	require.Contains(t, err.Error(), "refused")
	require.False(t, res.Ok)
	require.Equal(t, 0, res.StatusCode)
}

func TestRetryServerErrorExceptCodes(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls > 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusInsufficientStorage)
	}))
	defer ts.Close()

	req := gentleman.NewRequest()
	req.URL(ts.URL)
	req.Use(NewRetry(nil, ExceptCodes([]uint32{http.StatusServiceUnavailable})))

	res, err := req.Send()
	require.NoError(t, err)
	require.False(t, res.Ok)
	require.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	require.Equal(t, 2, calls)
}
