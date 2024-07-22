package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// You need to write a wrapper function that can be used to call the blocking function.
// The makeBlockingSyscall function can block for up to 10 seconds. We want to wait a maximum of 3 seconds.

func main() {
	ctx := context.Background()

	resultChan := runAsyncWithTimeout(ctx, makeBlockingSyscall, time.Second*3)

	start := time.Now()
	result := <-resultChan
	executionTime := time.Since(start)

	fmt.Printf("makeBlockingSyscall execution result: %ds\n", executionTime.Milliseconds()/1000)

	if result.Err != nil {
		fmt.Printf("makeBlockingSyscall failed with error: %v", result.Err)
	} else {
		fmt.Printf("makeBlockingSyscall succeeded with result: %s", result.Out)
	}
}

func makeBlockingSyscall() (string, error) {
	// Random delay from 0 to 10 seconds.
	time.Sleep(time.Second * rand.N[time.Duration](10))

	// Generates an error 20% of the time.
	if rand.N[int](10) < 2 {
		return "", errors.New("unexpected error")
	}

	return "success", nil
}

type asyncCallResult[Out any] struct {
	Out Out
	Err error
}

func runAsyncWithTimeout[Out any](
	ctx context.Context,
	fn func() (Out, error),
	timeout time.Duration,
) <-chan asyncCallResult[Out] {
	resultChan := make(chan asyncCallResult[Out])

	go func(ctx context.Context) {
		defer close(resultChan)

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		select {
		case <-ctx.Done():
			resultChan <- asyncCallResult[Out]{Err: errors.New("async call timeout")}
		case result := <-runNonblocking(fn):
			resultChan <- result
		}
	}(ctx)

	return resultChan
}

func runNonblocking[Out any](
	fn func() (Out, error),
) <-chan asyncCallResult[Out] {
	resultChan := make(chan asyncCallResult[Out])

	go func() {
		defer close(resultChan)

		out, err := fn()
		resultChan <- asyncCallResult[Out]{
			Out: out,
			Err: err,
		}
	}()

	return resultChan
}
