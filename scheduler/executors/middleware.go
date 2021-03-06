package executors

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	retry "gopkg.in/eapache/go-resiliency.v1/retrier"
	"gopkg.in/h2non/gentleman.v2/context"
	"gopkg.in/h2non/gentleman.v2/plugin"
)

const (
	// RetryTimes defines the default max amount of times to retry a request.
	RetryTimes = 3

	// RetryWait defines the default amount of time to wait before each retry attempt.
	RetryWait = 100 * time.Millisecond
)

var (
	// ErrServer stores the error when a server error happens.
	ErrServer = errors.New("retry: server response error")
)

// Retrier defines the required interface implemented by retry strategies.
type Retrier interface {
	Run(func() error) error
}

// EvalFunc represents the function interface for failed request evaluator.
type EvalFunc func(error, *http.Response, *http.Request) error

// Evaluator determines when a request failed in order to retry it,
// evaluating the error, response and optionally the original request.
//
// By default if will retry if an error is present or response status code is >= 500.
//
// You can override this function to use a custom evaluator function with additional logic.
var Evaluator = func(err error, res *http.Response, req *http.Request) error {
	if err != nil {
		return err
	}
	if res.StatusCode >= 500 || res.StatusCode == 429 {
		return ErrServer
	}
	return nil
}

func NewConstantBackoff(retryTimes int, retryWait time.Duration) Retrier {
	return retry.New(retry.ConstantBackoff(retryTimes, retryWait), nil)
}

func ExceptCodes(codes []uint32) EvalFunc {
	return func(err error, res *http.Response, req *http.Request) error {
		if err != nil {
			return err
		}

		for _, code := range codes {
			if res.StatusCode == int(code) {
				return nil
			}
		}

		if res.StatusCode >= 500 || res.StatusCode == 429 {
			return ErrServer
		}
		return nil
	}
}

// New creates a new retry plugin based on the given retry strategy.
func NewRetry(retrier Retrier, eval EvalFunc) plugin.Plugin {
	if retrier == nil {
		retrier = NewConstantBackoff(RetryTimes, RetryWait)
	}
	if eval == nil {
		eval = Evaluator
	}

	// Create retry new plugin
	plu := plugin.New()

	// Attach the middleware handler for before dial phase
	plu.SetHandler("before dial", func(ctx *context.Context, h context.Handler) {
		InterceptTransport(ctx, retrier, eval)
		h.Next(ctx)
	})

	return plu
}

// InterceptTransport is a middleware function handler that intercepts
// the HTTP transport based on the given HTTP retrier and context.
func InterceptTransport(ctx *context.Context, retrier Retrier, eval EvalFunc) {
	newTransport := &Transport{retrier, eval, ctx.Client.Transport, ctx}
	ctx.Client.Transport = newTransport
}

// Transport provides a http.RoundTripper compatible transport who encapsulates
// the original http.Transport and provides transparent retry support.
type Transport struct {
	retrier   Retrier
	evaluator EvalFunc
	transport http.RoundTripper
	context   *context.Context
}

// RoundTrip implements the required method by http.RoundTripper interface.
// Performs the network transport over the original http.Transport but providing retry support.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	res := t.context.Response

	// Cache all the body buffer
	buf, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return res, err
	}
	_ = req.Body.Close()

	// Transport request via retrier
	_ = t.retrier.Run(func() error {
		// Clone the http.Request for side effects free
		reqCopy := &http.Request{}
		*reqCopy = *req

		// Restore the cached body buffer
		reqCopy.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

		// Proxy to the original tranport round tripper
		res, err = t.transport.RoundTrip(reqCopy)
		return t.evaluator(err, res, req)
	})

	// Restore original http.Transport
	t.context.Client.Transport = t.transport

	return res, err
}
