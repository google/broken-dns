// Copyright 2022 Google LLC

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     https://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package jobs facilitates a method of parallelized workers like errgroup, but allows for circular pipelines
package jobs

import (
	"context"
	"sync"
)

// Job interface for data that can be used for a job
type Job interface {
}

// Jobs is an orchestrator of all threads and parallelism
type Jobs struct {
	jobChan  chan Job
	saveChan chan Job
	saveDone chan bool
	errChan  chan error
	wg       sync.WaitGroup
	saveWg   sync.WaitGroup
	jobCtx   context.Context
	ctx      context.Context
	cancel   context.CancelFunc
}

// Start creates a jobs instance
func Start(ctx context.Context) *Jobs {
	var j Jobs
	j.jobChan = make(chan Job, 1)
	j.saveChan = make(chan Job, 10)
	j.errChan = make(chan error)
	j.ctx = ctx
	j.jobCtx, j.cancel = context.WithCancel(j.ctx)
	return &j
}

// Go starts a new thread that consumes jobs
func (j *Jobs) Go(worker func(context.Context, chan Job) error) {
	go func() {
		err := worker(j.jobCtx, j.jobChan)
		if err != nil {
			//log.Printf("Go error: %s", err)
			if err != context.Canceled {
				// if multiple threads error and try to write at the same time
				// this can panic due to the first error closing the channel
				// for now, errChan is not closed to prevent this panic
				j.errChan <- err
			}
		}
	}()
}

// Add add a work item
func (j *Jobs) Add(jobs ...Job) {
	j.wg.Add(len(jobs))
	go func() {
		for _, w := range jobs {
			// add jobs
			j.jobChan <- w
		}
	}()
}

// Done should be called by the worker threads when they finish a job
func (j *Jobs) Done(w Job) {
	j.Save(w)
	j.wg.Done()
}

// Save a work item without working on it, useful for metadata
func (j *Jobs) Save(w Job) {
	j.saveWg.Add(1)
	j.saveChan <- w
}

// Wait blocks until all jobs are done or there is an error
func (j *Jobs) Wait() error {
	wgDone := make(chan bool)

	// goroutine to wait until WaitGroup is done
	go func() {
		j.wg.Wait()
		j.cancel()
		close(j.saveChan) // no more entries to be saved
		j.saveWg.Wait()   // wait for saving to finish
		close(wgDone)
	}()

	// Wait until either WaitGroup is done or an error is received through the channel
	select {
	case <-wgDone:
		// carry on
		break
	case err := <-j.errChan:
		// possible bug here, when we call cancel, all workers will return and error (context canceled)
		// and once errChan is closed this will cause a panic.
		// this is fixed by having Go() workers not return the context.Canceled error
		j.cancel()
		//close(j.errChan) // don't close errChan to allow multiple errors to be sent without causing a panic
		return err
	}

	// wait for save worker to finish and check for error
	select {
	case <-j.saveDone:
		// carry on
		break
	case err := <-j.errChan:
		// all worksers should have already finished by now, so no need to cancel, but calling just in case
		// if somehow a worker is canceld here, it will likely return an error and cause a panic when we close errChan
		j.cancel()
		// all workers should be done by now so safe to cancel
		close(j.errChan)
		return err
	}
	return nil
}

func (j *Jobs) SaveGo(saver func(context.Context, chan Job, *sync.WaitGroup) error) {
	j.saveDone = make(chan bool)
	go func() {
		err := saver(j.ctx, j.saveChan, &j.saveWg)
		if err != nil {
			//log.Printf("Go error: %s", err)
			if err != context.Canceled {
				// if multiple threads error and try to write at the same time
				// this can panic due to the first error closing the channel
				// for now, errChan is not closed to prevent this panic
				j.errChan <- err
				return // saveDone is not set or closed to ensure that the error is picked up
			}
		}
		// notify Wait() that saving is done
		j.saveDone <- true
		close(j.saveDone)
	}()
}
