package riprovare

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Retryable is a function that can be retried. If a non-nil error value is
// returned the function will be retried based on the RetryPolicy.
type Retryable func() error

// RetryPolicy is function type that returns a boolean indicating if operations
// should continue retrying. An error is accepted that allows for the error value
// to be inspected. Optionally retries can be abandoned or continue depending on
// the error value.
type RetryPolicy func(error) bool

// OnErrorFunc is a function type that is invoked when an error occurs which provides
// a hook to log errors, capture metrics, etc.
type OnErrorFunc func(error)

// SimpleRetryPolicy is a RetryPolicy that retries the max attempts with no delay
// between retries.
func SimpleRetryPolicy(attempts int) RetryPolicy {
	return func(err error) bool {
		// If the error is from the context being canceled there is no reason
		// to continue retrying
		if errors.Is(err, context.Canceled) {
			return false
		}
		if attempts--; attempts > 0 {
			return true
		}
		return false
	}
}

// FixedRetryPolicy returns a RetryPolicy that retries the max attempts delaying
// the provided fixed duration between attempts.
func FixedRetryPolicy(attempts int, delay time.Duration) RetryPolicy {
	return func(err error) bool {
		// If the error is from the context being canceled there is no reason
		// to continue retrying
		if errors.Is(err, context.Canceled) {
			return false
		}
		if attempts--; attempts > 0 {
			time.Sleep(delay)
			return true
		}
		return false
	}
}

// ExponentialBackoffRetryPolicy is a RetryPolicy that retries the max attempts
// with a delay between each retry. After each attempt the delay duration is doubled
// +/- 25% jitter.
func ExponentialBackoffRetryPolicy(attempts int, initialDelay time.Duration) RetryPolicy {
	delay := initialDelay
	return func(err error) bool {
		// If the error is from the context being canceled there is no reason
		// to continue retrying
		if errors.Is(err, context.Canceled) {
			return false
		}
		if attempts--; attempts > 0 {
			time.Sleep(delay)
			delay = exponential(delay)
			return true
		}
		return false
	}
}

// Option allows additional configuration of the retries.
type Option func(r *retry)

// ErrorHook adds a callback when an error occurs but before the next retry.
// This allows for the user of this package to capture errors or logging,
// metrics, etc.
func ErrorHook(fn OnErrorFunc) Option {
	// Protect against illegal use of API, if someone does this all hope is lost.
	// Technically letting this pass wouldn't cause a panic at runtime because the
	// OnErrorFunc is only invoked if it is non-nil, but passing the ErrorHook option
	// to Retry with a nil can be nothing but a programmer error because well ... it
	// makes no sense.
	if fn == nil {
		panic(fmt.Errorf("illegal use of api, cannot invoke a nil function"))
	}
	return func(r *retry) {
		r.onError = fn
	}
}

// Retry invokes a Retryable and retries according to the provided RetryPolicy.
// Once all attempts have been exhausted this function will return an
// UnrecoverableError.
//
// A zero-value/nil RetryPolicy or Retryable will cause a panic.
func Retry(policy RetryPolicy, fn Retryable, opts ...Option) error {
	if policy == nil {
		panic(fmt.Errorf("illegal use of api: cannot operate on nil RetryPolicy"))
	}
	if fn == nil {
		panic(fmt.Errorf("illegal use of api: cannot invoke nil function"))
	}
	r := &retry{
		fn:     fn,
		policy: policy,
	}

	for _, opt := range opts {
		opt(r)
	}
	return r.do()
}

type retry struct {
	policy  RetryPolicy
	fn      Retryable
	onError OnErrorFunc
}

func (r retry) do() error {
	if err := r.fn(); err != nil {
		if r.onError != nil {
			r.onError(err)
		}
		if r.policy(err) {
			return r.do()
		}
		return UnrecoverableError{Err: err}
	}
	return nil
}

type UnrecoverableError struct {
	Err error
}

func (u UnrecoverableError) Error() string {
	return fmt.Sprintf("max retries exceeded: %s", u.Err)
}

func exponential(d time.Duration) time.Duration {
	d *= 2
	jitter := rand.Float64() + 0.25
	d = time.Duration(int64(float64(d.Nanoseconds()) * jitter))
	return d
}
