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
	"log"
	"sync"
)

type LameStats struct {
	Total    uint
	Lame     uint
	Problems uint
}

func (s *LameStats) String() string {
	return fmt.Sprintf("STATS: %d/%d lame delegations and %d problems", s.Lame, s.Total, s.Problems)
}

func saver(ctx context.Context, saveChan chan jobs.Job, wg *sync.WaitGroup) error {
	var stats LameStats
	defer func() {
		fmt.Println(stats.String())
	}()
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			return err
		case workItem, ok := <-saveChan:
			if !ok {
				// nothing left to save
				return nil
			}
			switch d := workItem.(type) {
			case *nameWork:
				stats.Total++
				if d.Lame {
					stats.Lame++
				}
				stats.Problems += d.Problems
			default:
				log.Fatalf("ERROR: saver: don't know about type %T!\n%+v\n", v, d)
			}
			// print to stdout as json for testing
			wg.Done()
		}
	}
}
