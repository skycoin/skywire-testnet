// Package testhelpers provides helpers for testing.
package testhelpers

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Timeout defines timeout for NoErrorWithinTimeout
var Timeout = 5 * time.Second

// joinErrChannels multiplexes several error channels
func joinErrChannels(errChans []<-chan error) chan error {
	joinedCh := make(chan error)
	for _, ch := range errChans {
		go func(errCh <-chan error) {
			err := <-errCh
			if err != nil {
				joinedCh <- err
			}
		}(ch)
	}
	return joinedCh
}

// NoErrorWithinTimeout tries to read an error from error channel within timeout and returns it.
// If timeout exceeds, nil value is returned.
func NoErrorWithinTimeout(ch <-chan error) error {
	select {
	case err := <-ch:
		return err
	case <-time.After(Timeout):
		return nil
	}
}

// NoErrorWithinTimeoutN tries to read an error from error channels within timeout and returns it.
// If timeout exceeds, nil value is returned.
func NoErrorWithinTimeoutN(errChans ...<-chan error) error {
	return NoErrorWithinTimeout(joinErrChannels(errChans))
}

func formatName(name string) string {
	parts := strings.Split(name, "/")
	return fmt.Sprintf("[%v]", parts[len(parts)-1])
	// return strings.Join(parts[len(parts)-3:], ".")
}

// GetCallerN return name of nth-caller
func GetCallerN(skip int) string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return formatName(frame.Function)
}

// GetCaller returns caller name
func GetCaller() string {
	return GetCallerN(3)
}

// Trace returns caller name with attached label
func Trace(label string) string {
	return fmt.Sprintf("%v %v", GetCallerN(3), label)
}

// CallerDepth return depth of function
func CallerDepth() int {
	pc := make([]uintptr, 15)
	return runtime.Callers(9, pc)
}

// GetCallers return names of caller functions
func GetCallers(skip int) []string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(skip, pc)
	frames := runtime.CallersFrames(pc[:n])
	var callers []string
	for frame, next := frames.Next(); next; frame, next = frames.Next() {
		callers = append(callers, formatName(frame.Function))
	}
	return callers
}
