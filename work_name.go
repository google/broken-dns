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
	"log"
)

type nameWork struct {
	Name     string
	Lame     bool
	Problems uint
}

func processName(ctx context.Context, w *nameWork) error {
	v("processName: %q", w.Name)
	var err error

	// get labels
	labels := SplitDomainNameWithParent(w.Name)

	action := false

	servers := rootServers // first iteration will use the ROOT nameservers

	// iterate backwards from TLD to domain
	for i := len(labels) - 1; i >= 0; i-- {
		// starting from the tld, work our way down to see what is in the cache
		addFun, first := seen.AddCheck(labels[i])
		if first {
			action = true
			v("checking: (%q) %q", w.Name, labels[i])
			result, err := queryNSParallel(labels[i], servers)
			if err != nil {
				err2 := addFun(nil, err)
				if err2 != nil {
					log.Printf("ERROR on addFun() while handling another error: %s", err2.Error())
				}
				return err
			}

			// testing
			v("got result for (%q) %q: %+v", w.Name, labels[i], result.String())

			// set next iteration of loop to use the servers we just got (unless we got authoritative answers with no servers)
			servers = result.NS
			if len(result.NS) == 0 {
				v("no nameservers found for (%q) %q", w.Name, labels[i])
				// in this case, sometimes the parent domain's nameservers are still authoritative
				// if no nameservers but I there ARE authoritative answers, iterate again on the same server list (of authoritative only?)
				authoritativeServers := result.GetAuthorativeNS()
				if len(authoritativeServers) > 0 {
					v("using parents authoritative nameservers for %q: %v", labels[i], authoritativeServers)
					servers = authoritativeServers
				}
			} // else {
			// 	// set next iteration of loop to use the servers we just got
			// 	servers = result.NS
			// }

			// do add (get data and save back to cache)
			err = addFun(servers, nil)
			if err != nil {
				return err
			}

			w.Problems += checkEqualResultResponse(result)

			w.Lame = checkLame(result)
			if w.Lame {
				w.Problems++
			}

			if i == 0 { // the full domain name, not a parent
				// check for expected NS
				w.Problems += checkExpectedNS(result)
			}

		} else {
			// get servers from cache
			v("waiting for cache to be populated for (%q)%q", w.Name, labels[i])
			servers, err = seen.GetWait(labels[i])
			if err != nil {
				return err
			}
			v("cache response for (%q) %q: %v", w.Name, labels[i], servers)
		}

		// here I can do a test if desired on every iteration of each label.
		// to not duplicate tests, most are done in the "first" section above
	}

	if !action {
		v("no action taken for %q, possible dup?", w.Name)
	}

	return nil
}
