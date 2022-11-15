package riprovare

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry_Success(t *testing.T) {
	var result int
	err := Retry(SimpleRetryPolicy(3), func() error {
		result = 5
		return nil
	})

	assert.Equal(t, 5, result)
	assert.NoError(t, err)
}

func TestRetry_Failure(t *testing.T) {
	attempts := 0
	err := Retry(SimpleRetryPolicy(3), func() error {
		attempts++
		return fmt.Errorf("oh snap this broke")
	})

	unrecoverable := &UnrecoverableError{}
	assert.Equal(t, 3, attempts)
	assert.Error(t, err)
	assert.ErrorAs(t, err, unrecoverable)
}

func TestRetry_Recovered(t *testing.T) {
	attempts := 0
	err := Retry(SimpleRetryPolicy(3), func() error {
		attempts++
		if attempts == 3 {
			return nil
		}
		return fmt.Errorf("oh snap this broke")
	})

	assert.Equal(t, 3, attempts)
	assert.NoError(t, err)
}

func TestFixedRetryPolicy(t *testing.T) {
	counter := 0
	start := time.Now()
	policy := FixedRetryPolicy(3, time.Second*1)
	for i := 0; i <= 2; i++ {
		counter++
		if !policy(nil) {
			break
		}
	}
	assert.Equal(t, 3, counter)
	assert.GreaterOrEqual(t, time.Since(start), 2*time.Second)
}

func TestFixedRetryPolicy_ContextCanceled(t *testing.T) {
	counter := 0
	policy := FixedRetryPolicy(3, time.Second*1)
	for i := 0; i <= 2; i++ {
		counter++
		if !policy(context.Canceled) {
			break
		}
	}
	assert.Equal(t, 1, counter)
}

func TestExponentialBackoffRetryPolicy(t *testing.T) {
	counter := 0
	lastDuration := time.Duration(0)
	policy := ExponentialBackoffRetryPolicy(3, 1*time.Second)
	for i := 0; i <= 2; i++ {
		counter++
		start := time.Now()
		if !policy(nil) {
			break
		}
		duration := time.Since(start)
		assert.Greater(t, duration, lastDuration)
		lastDuration = duration
	}
	assert.Equal(t, 3, counter)
}

func TestExponentialBackoffRetryPolicy_ContextCanceled(t *testing.T) {
	counter := 0
	policy := ExponentialBackoffRetryPolicy(3, 1*time.Second)
	for i := 0; i <= 2; i++ {
		counter++
		if !policy(context.Canceled) {
			break
		}
	}
	assert.Equal(t, 1, counter)
}

func TestRetry_ErrorHook(t *testing.T) {
	counter := 0
	hookCounter := 0

	hook := OnErrorFunc(func(err error) {
		hookCounter++
	})

	err := Retry(SimpleRetryPolicy(3), func() error {
		counter++
		return fmt.Errorf("oh snap this broke")
	}, ErrorHook(hook))
	assert.Error(t, err)
	assert.Equal(t, 3, counter)
	assert.Equal(t, 3, hookCounter)
}
