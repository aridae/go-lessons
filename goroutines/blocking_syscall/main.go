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

	start := time.Now()
	res, err := runWithTimeout(ctx, makeBlockingSyscall, time.Second*3)
	executionTime := time.Since(start)

	fmt.Printf("makeBlockingSyscall execution time: %ds\n", executionTime.Milliseconds()/1000)

	if err != nil {
		fmt.Printf("makeBlockingSyscall failed with error: %v", err)
	} else {
		fmt.Printf("makeBlockingSyscall succeeded with result: %s", res)
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

func runWithTimeout[Out any](
	ctx context.Context,
	fn func() (Out, error),
	timeout time.Duration,
) (Out, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var out Out
	select {
	case <-ctx.Done():
		return out, errors.New("async call timeout")
	case result := <-runNonblocking(fn):
		return result.Out, result.Err
	}
}

type asyncCallResult[Out any] struct {
	Out Out
	Err error
}

func runNonblocking[Out any](
	fn func() (Out, error),
) <-chan asyncCallResult[Out] {
	resultChan := make(chan asyncCallResult[Out], 1)

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
