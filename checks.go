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

import "github.com/miekg/dns"

func checkEqualResultResponse(r *queryGroup) uint {
	totalServers := len(r.NS)
	var found uint = 0

	for server := range r.Results {
		if r.Results[server].Err != nil {
			finding("ERROR server: %q @%s: %s", r.Domain, server, r.Results[server].Err)
			found++
		} else {
			// only check for different responses if the query did not error
			serverResponses := len(r.Results[server].NS)
			if serverResponses != totalServers {
				missing := ExtraStrings(r.NS, r.Results[server].NS)
				finding("varying responses: expected %d, got %d, for %q @%s. missing: %v", totalServers, serverResponses, r.Domain, server, missing)
				found++
			}
		}
	}
	return found
}

func checkLame(q *queryGroup) bool {
	//v("checkLame(%q)", q.Domain)
	r, err := queryNSParallel(q.Domain, q.NS)
	lame := false
	if err != nil {
		finding("ERROR querying authoritative: %s %s", q.Domain, err)
		return true
	}
	v("checkLame(%q) query result: \n\t%+v", q.Domain, r.String())

	if !StringArrayEquals(q.NS, r.NS) {
		lame = true
		finding("unexpected difference in nameservers: domain: %q expected %d: %v, got %d: %v", q.Domain, len(q.NS), q.NS, len(r.NS), r.NS)
		extra := ExtraStrings(r.NS, q.NS)
		if len(extra) > 0 {
			finding("> extra nameservers returned by authoritative NS: %q: %v", q.Domain, ExtraStrings(r.NS, q.NS))
			// TODO extra nameservers can also be lame, check
		}
	}

	// check that the authoritative bit is set
	for nameserver := range r.Results {
		if !r.Results[nameserver].Authoritative {
			lame = true
			finding("lame delegation: %q is not authoritative for %q", nameserver, r.Domain)
		}
	}
	return lame
}

// TODO this can be made more efficient, lots of redundant checks
// test of returned ns are withing the expected set
func checkExpectedNS(r *queryGroup) uint {
	var found uint = 0
	if len(expectedNameServers) > 0 {
		// iterate over all found NSd
		for _, ns := range r.NS {
			problem := true
			// iterate over all expected NS
			for _, expectedNS := range expectedNameServers {
				if dns.IsSubDomain(expectedNS, ns) {
					problem = false
				}
			}
			if problem {
				finding("unexpected nameserver: %q NS %q", r.Domain, ns)
				found++
			}
		}
	}
	return found
}
