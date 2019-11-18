package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker(t *testing.T) {
	cb := New(3, time.Minute)
	taskCallCount := 0
	fallbackCallCount := 0
	task := func() error {
		taskCallCount++
		return errors.New("some error")
	}
	fallback := func() error {
		fallbackCallCount++
		return nil
	}

	cb.Handle(task, fallback)
	cb.Handle(task, fallback)
	cb.Handle(task, fallback)
	cb.Handle(task, fallback)
	if taskCallCount != 3 {
		t.Fatalf("taskCallCount expected:%d,got:%d", 3, taskCallCount)
	}

	if fallbackCallCount != 4 {
		t.Fatalf("fallbackCallCount expected:%d,got:%d", 4, fallbackCallCount)
	}

}
