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
	"github.com/miekg/dns"
)

func stringMapToArrayKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	return out
}

// SplitDomainNameWithParent splits a name string into it's labels with each parent.
func SplitDomainNameWithParent(s string) (labels []string) {
	if s == "" {
		return nil
	}
	fqdnEnd := 0 // offset of the final '.' or the length of the name
	idx := dns.Split(s)
	begin := 0
	if dns.IsFqdn(s) {
		fqdnEnd = len(s) - 1
	} else {
		fqdnEnd = len(s)
	}

	switch len(idx) {
	case 0:
		return nil
	case 1:
		// no-op
	default:
		for _, end := range idx[1:] {
			labels = append(labels, s[begin:fqdnEnd])
			begin = end
		}
	}

	return append(labels, s[begin:fqdnEnd])
}

func StringArrayEquals(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func StringArrayToMap(a []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range a {
		m[s] = true
	}
	return m
}

// ExtraStrings takes two string arrays and returns a slice of the strings in a not in b
func ExtraStrings(a, b []string) []string {
	c := make([]string, 0, len(a)/2)
	bMap := StringArrayToMap(b)
	for _, s := range a {
		if !bMap[s] {
			c = append(c, s)
		}
	}
	return c
}
