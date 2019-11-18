package circuitbreaker

import (
	"sync/atomic"
	"time"
)

type status uint32

const (
	statusClosed status = iota
	statusHalfOpen
	statusOpen
)

// Task 任务
type Task func() error

// Fallback 备用
type Fallback func() error

// CircuiBreaker 断路器
type CircuiBreaker struct {
	maxErrorCount     uint32
	retryInterval     time.Duration
	curErrorCount     uint32
	lastOpenTimestamp int64
	retrying          uint32
	status            status
}

// New 初始化断路器
func New(maxErrorCount uint32, retryInterval time.Duration) *CircuiBreaker {
	if retryInterval <= 0 {
		panic("zero or negative retryInterval")
	}
	return &CircuiBreaker{
		maxErrorCount: maxErrorCount,
		retryInterval: retryInterval,
	}
}

// Handle 处理任务
func (cb *CircuiBreaker) Handle(task Task, fallback Fallback) error {
	switch cb.getStatus() {
	case statusOpen:
		return cb.handleOpen(fallback)
	case statusHalfOpen:
		return cb.handleHalfOpen(task, fallback)
	case statusClosed:
		return cb.handleClosed(task, fallback)
	default:
		panic("unexpected status")
	}
}

func (cb *CircuiBreaker) handleHalfOpen(task Task, fallback Fallback) error {
	if !cb.trySetRetrying() {
		return cb.handleOpen(fallback)
	}
	if err := cb.handleNormal(task, fallback, false); err != nil {
		return err
	}
	cb.setClosed()
	cb.setRetryingFinish()
	return nil
}

func (cb *CircuiBreaker) handleNormal(task Task, fallback Fallback, setOpen bool) error {
	if err := task(); err == nil {
		return nil
	}
	cb.addError()
	if setOpen {
		cb.trySetOpen()
	}
	if err := fallback(); err != nil {
		cb.addError()
		if setOpen {
			cb.trySetOpen()
		}
		return err
	}
	return nil
}

func (cb *CircuiBreaker) handleOpen(fallback Fallback) error {
	cb.trySetHalfOpen()
	if err := fallback(); err != nil {
		cb.addError()
		return err
	}
	return nil
}

func (cb *CircuiBreaker) handleClosed(task Task, fallback Fallback) error {
	if err := cb.handleNormal(task, fallback, true); err != nil {
		return err
	}
	return nil
}

func (cb *CircuiBreaker) getStatus() status {
	return status(atomic.LoadUint32((*uint32)(&cb.status)))
}

func (cb *CircuiBreaker) trySetRetrying() bool {
	return atomic.CompareAndSwapUint32(&cb.retrying, 0, 1)
}

func (cb *CircuiBreaker) setRetryingFinish() {
	atomic.StoreUint32(&cb.retrying, 0)
}

func (cb *CircuiBreaker) trySetHalfOpen() {
	lastOpenTimestamp := atomic.LoadInt64(&cb.lastOpenTimestamp)
	now := time.Now()
	if now.Sub(time.Unix(lastOpenTimestamp, 0)) > cb.retryInterval {
		atomic.CompareAndSwapUint32((*uint32)(&cb.status), uint32(statusOpen), uint32(statusHalfOpen))
	}
}

func (cb *CircuiBreaker) setClosed() {
	atomic.StoreUint32((*uint32)(&cb.status), uint32(statusClosed))
}

func (cb *CircuiBreaker) trySetOpen() {
	curErrorCount := atomic.LoadUint32(&cb.curErrorCount)
	if curErrorCount >= cb.maxErrorCount {
		atomic.StoreUint32((*uint32)(&cb.status), uint32(statusOpen))
		atomic.StoreUint32(&cb.curErrorCount, 0)
	}
}

func (cb *CircuiBreaker) addError() {
	atomic.AddUint32(&cb.curErrorCount, 1)
}
