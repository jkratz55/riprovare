# Riprovare

Riprovare is a package providing a retry mechanism for Go. The name riprovare comes from the Italian word for retry (or at least that is what Google Translate claims). The goal for this package is to provide simple but configurable API for handling retry logic.

## Getting Riprovare

```shell
go get -u github.com/jkratz55/riprovare
```

## Usage

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jkratz55/riprovare"
)

func main() {

	req, err := http.NewRequest(http.MethodGet, "https://google.com", nil)
	if err != nil {
		panic(err)
	}

	var result string
	err = riprovare.Retry(riprovare.FixedRetryPolicy(3, time.Second*1), func() error {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s %s returned non success http status code %d", http.MethodGet,
				"https://google.com", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		result = string(body)
		return nil
	})

	if err != nil {
		panic(err)
	}
	fmt.Println(result)
}
```

The heart of Riprovare is the Retry function. It accepts a RetryPolicy, a closure, and optionally options to further configure the behavior. The RetryPolicy controls the policy for retries by returning a boolean indicating if the closure should be retried if it returned a non-nil error value. Riprovare comes with three built in retry policies.

* SimpleRetryPolicy - Attempts to execute the closure up to the specified attempts.
* FixedRetryPolicy - Attempts to execute the closure up to the specified attempts with a fixed delay between each attempt.
* ExponentialBackoffRetryPolicy - Attempts to execute the closure up to the specified attempts with exponential backoff and 25% jitter. 

The built-in retry policies may not cover all cases, but you can always provide your own RetryPolicy as it's simply a function that accepts an error and returns a boolean. Since a RetryPolicy accepts an error a custom RetryPolicy can inspect the error and decide to retry certain types of error but not others. 

## Error Handling

By default, the Retry function will swallow errors until all the retries have been exceeded, and then it will return an UnrecoverableError which contains the root error. However, often times you may want to either log errors, or capture metrics on failed attempts even though there are retries remaining. Technically, this could be accomplished within the closure passed to Retry, but Riprovare offers a more elegant way to handle this. The Retry function accepts variadic Options to further customize the behavior of retries. One such option is ErrorHook which accepts a func(error) and is invoked whenever the closure returns a non-nil error.

## Contributions

Contributions are welcome, but it's always a good idea to open an issue first as to not waste time on something that would never be merged. 