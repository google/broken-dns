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

package main

import (
	"context"
	"fmt"
	"lame-dns/jobs"
)

func worker(ctx context.Context, workChan chan jobs.Job) error {
	for {
		select {
		case <-ctx.Done():
			// we expect the context to be canceled when all the work is done
			err := ctx.Err()
			return err
		case workItem := <-workChan:
			var err error
			//v("worker: %+v", workItem)
			switch v := workItem.(type) {
			case *nameWork:
				err = processName(ctx, v)
			default:
				return fmt.Errorf("don't know about type: %T", v)
			}
			work.Done(workItem)
			if err != nil {
				return fmt.Errorf("error on %w: %v", err, workItem)
			}
		}
	}
}
