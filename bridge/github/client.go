package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/shurcooL/githubv4"
)

var _ Client = &githubv4.Client{}

// Client is an interface conforming with githubv4.Client
type Client interface {
	Mutate(context.Context, interface{}, githubv4.Input, map[string]interface{}) error
	Query(context.Context, interface{}, map[string]interface{}) error
}

// rateLimitHandlerClient wrapps the Github client and adds improved error handling and handling of
// Github's GraphQL rate limit.
type rateLimitHandlerClient struct {
	sc Client
}

func newRateLimitHandlerClient(httpClient *http.Client) *rateLimitHandlerClient {
	return &rateLimitHandlerClient{sc: githubv4.NewClient(httpClient)}
}

type RateLimitingEvent struct {
	msg string
}

// mutate calls the github api with a graphql mutation and for each rate limiting event it sends an
// export result.
func (c *rateLimitHandlerClient) mutate(ctx context.Context, m interface{}, input githubv4.Input, vars map[string]interface{}, out chan<- core.ExportResult) error {
	// prepare a closure for the mutation
	mutFun := func(ctx context.Context) error {
		return c.sc.Mutate(ctx, m, input, vars)
	}
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			select {
			case <-ctx.Done():
				return
			case out <- core.NewExportRateLimiting(e.msg):
			}
		}
	}()
	return c.callAPIAndRetry(mutFun, ctx, limitEvents)
}

// queryWithLimitEvents calls the github api with a graphql query and it sends rate limiting events
// to a given channel of type RateLimitingEvent.
func (c *rateLimitHandlerClient) queryWithLimitEvents(ctx context.Context, query interface{}, vars map[string]interface{}, limitEvents chan<- RateLimitingEvent) error {
	// prepare a closure fot the query
	queryFun := func(ctx context.Context) error {
		return c.sc.Query(ctx, query, vars)
	}
	return c.callAPIAndRetry(queryFun, ctx, limitEvents)
}

// queryWithImportEvents calls the github api with a graphql query and it sends rate limiting events
// to a given channel of type ImportEvent.
func (c *rateLimitHandlerClient) queryWithImportEvents(ctx context.Context, query interface{}, vars map[string]interface{}, importEvents chan<- ImportEvent) error {
	// forward rate limiting events to channel of import events
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			select {
			case <-ctx.Done():
				return
			case importEvents <- e:
			}
		}
	}()
	return c.queryWithLimitEvents(ctx, query, vars, limitEvents)
}

// queryPrintMsgs calls the github api with a graphql query and it prints for ever rate limiting
// event a message to stdout.
func (c *rateLimitHandlerClient) queryPrintMsgs(ctx context.Context, query interface{}, vars map[string]interface{}) error {
	// print rate limiting events directly to stdout.
	limitEvents := make(chan RateLimitingEvent)
	defer close(limitEvents)
	go func() {
		for e := range limitEvents {
			fmt.Println(e.msg)
		}
	}()
	return c.queryWithLimitEvents(ctx, query, vars, limitEvents)
}

// callAPIAndRetry calls the Github GraphQL API (inderectely through callAPIDealWithLimit) and in
// case of error it repeats the request to the Github API. The parameter `apiCall` is intended to be
// a closure containing a query or a mutation to the Github GraphQL API.
func (c *rateLimitHandlerClient) callAPIAndRetry(apiCall func(context.Context) error, ctx context.Context, events chan<- RateLimitingEvent) error {
	var err error
	if err = c.callAPIDealWithLimit(apiCall, ctx, events); err == nil {
		return nil
	}
	// failure; the reason may be temporary network problems or internal errors
	// on the github servers. Internal errors on the github servers are quite common.
	// Retry
	retries := 3
	for i := 0; i < retries; i++ {
		// wait a few seconds before retry
		sleepTime := time.Duration(8*(i+1)) * time.Second
		timer := time.NewTimer(sleepTime)
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
			err = c.callAPIDealWithLimit(apiCall, ctx, events)
			if err == nil {
				return nil
			}
		}
	}
	return err
}

// callAPIDealWithLimit calls the Github GraphQL API and if the Github API returns a rate limiting
// error, then it waits until the rate limit is reset and it repeats the request to the API. The
// parameter `apiCall` is intended to be a closure containing a query or a mutation to the Github
// GraphQL API.
func (c *rateLimitHandlerClient) callAPIDealWithLimit(apiCall func(context.Context) error, ctx context.Context, events chan<- RateLimitingEvent) error {
	qctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	// call the function fun()
	err := apiCall(qctx)
	if err == nil {
		return nil
	}
	// matching the error string
	if strings.Contains(err.Error(), "API rate limit exceeded") {
		// a rate limit error
		qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		// Use a separate query to get Github rate limiting information.
		limitQuery := rateLimitQuery{}
		if err := c.sc.Query(qctx, &limitQuery, map[string]interface{}{}); err != nil {
			return err
		}
		// Get the time when Github will reset the rate limit of their API.
		resetTime := limitQuery.RateLimit.ResetAt.Time
		resetTime = resetTime.Add(time.Second * 16)
		msg := fmt.Sprintf(
			"Github GraphQL API rate limit. This process will sleep until %s.",
			resetTime.String(),
		)
		// Send message about rate limiting event.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- RateLimitingEvent{msg}:
		}
		// Pause current goroutine
		timer := time.NewTimer(time.Until(resetTime))
		select {
		case <-ctx.Done():
			stop(timer)
			return ctx.Err()
		case <-timer.C:
		}
		// call the function apiCall() again
		qctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
		err = apiCall(qctx)
		return err // might be nil
	} else {
		return err
	}
}

func stop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
