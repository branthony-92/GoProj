package asyncprocessor

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type ProcessCallback func(any) error
type ProcessCancelFunc func()
type ProcessFunc func() any

type AsyncProcessor interface {
	Queue(ProcessFunc)
	DataChan() chan<- any
	Launch()
	Wait() error
	Cancel()
}

type options struct {
	timeout    time.Duration
	callback   func(any) error
	bufferSize uint
}

type Option func(*options)

func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

func WithCallback(callback func(any) error) Option {
	return func(o *options) {
		o.callback = callback
	}
}

func WithBufferSize(size uint) Option {
	return func(o *options) {
		o.bufferSize = size
	}
}

type asyncProcessor struct {
	opts options

	dataCh chan any
	doneCh chan struct{}

	remainingJobs uint
	lastErr       error

	mutex sync.Mutex

	workQueue []ProcessFunc
}

func defaultOptions() options {
	return options{
		timeout: 5 * time.Second,
		callback: func(any) error {
			return nil
		},
		bufferSize: 1,
	}
}

func NewProcess(opts ...Option) AsyncProcessor {

	o := defaultOptions()

	for _, opt := range opts {
		opt(&o)
	}

	proc := asyncProcessor{
		opts:      o,
		doneCh:    make(chan struct{}),
		dataCh:    make(chan any, o.bufferSize),
		workQueue: make([]ProcessFunc, 0),
	}
	go proc.startProcessing()

	return &proc
}

func (proc *asyncProcessor) DataChan() chan<- any {
	return proc.dataCh
}

func (proc *asyncProcessor) Queue(fn ProcessFunc) {
	proc.workQueue = append(proc.workQueue, fn)
}

func (proc *asyncProcessor) Launch() {
	for _, fn := range proc.workQueue {
		proc.remainingJobs++
		go func(fn ProcessFunc) {
			val := fn()
			proc.dataCh <- val
		}(fn)
	}
	proc.workQueue = make([]ProcessFunc, 0)
}

func (proc *asyncProcessor) Wait() error {
	if proc.lastErr != nil {
		return proc.lastErr
	}
	<-proc.doneCh
	return proc.lastErr
}

func (proc *asyncProcessor) Cancel() {
	proc.doneCh <- struct{}{}
}

func (proc *asyncProcessor) startProcessing() {
	defer proc.Cancel()

	for {
		select {
		case d := <-proc.dataCh:
			err := proc.opts.callback(d)
			proc.remainingJobs--
			if err != nil {
				proc.lastErr = fmt.Errorf("callback error: %s", err.Error())
			}
			if proc.remainingJobs == 0 {
				return
			}
		case <-time.After(proc.opts.timeout):
			proc.lastErr = errors.New("proccess error: timeout")
			return
		}
	}
}

func doWork(val int) int {
	val *= val
	return val
}
