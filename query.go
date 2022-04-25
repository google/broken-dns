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
	"net"
	"sort"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/sync/errgroup"
)

// https://www.iana.org/domains/root/servers
var rootServers = []string{
	"a.root-servers.net",
	"b.root-servers.net",
	"c.root-servers.net",
	"d.root-servers.net",
	"e.root-servers.net",
	"f.root-servers.net",
	"g.root-servers.net",
	"h.root-servers.net",
	"i.root-servers.net",
	"j.root-servers.net",
	"k.root-servers.net",
	"l.root-servers.net",
	"m.root-servers.net",
}

const (
	dnsTimeout = time.Second * 10
	dnsRetry   = 3
)

var dnsClient = dns.Client{
	Timeout: dnsTimeout,
}

type queryResult struct {
	Err           error
	Authoritative bool
	NS            []string
}

func (r *queryResult) String() string {
	return fmt.Sprintf("Err: %v, AA: %t, NS: %+v", r.Err, r.Authoritative, r.NS)
}

type queryGroup struct {
	Domain  string
	Results map[string]*queryResult
	NS      []string
}

func (g *queryGroup) GetAuthorativeNS() []string {
	out := make([]string, 0, len(g.Results))
	for server := range g.Results {
		if g.Results[server].Authoritative {
			out = append(out, server)
		}
	}
	return out
}

func (g *queryGroup) String() string {
	out := fmt.Sprintf("Domain: %q\n", g.Domain)
	out += fmt.Sprintf("\tAllNS: %v\n", g.NS)
	for server := range g.Results {
		out += fmt.Sprintf("\t\t%q: %s\n", server, g.Results[server].String())
	}
	return out
}

func (g *queryGroup) allNS() []string {
	m := make(map[string]bool)

	for server := range g.Results {
		for _, ns := range g.Results[server].NS {
			m[ns] = true
		}
	}

	out := stringMapToArrayKeys(m)
	sort.Strings(out)

	return out
}

func queryNSParallel(domain string, servers []string) (*queryGroup, error) {
	domain = dns.Fqdn(domain)
	g, _ := errgroup.WithContext(context.Background())
	results := make([]*queryResult, len(servers))

	for i, server := range servers {
		i, server := i, server // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			ns, auth, err := queryNSServer(server, domain)
			results[i] = &queryResult{
				Err:           err,
				NS:            ns,
				Authoritative: auth,
			}
			if err != nil {
				v("error on queryNSServer(%q, %q): %v", server, domain, err)
				// don't return the error so that all queries capture their responses
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	result := &queryGroup{
		Domain:  cleanDomain(domain),
		Results: make(map[string]*queryResult),
	}

	// add results from array to map
	for i := range results {
		result.Results[servers[i]] = results[i]
	}

	result.NS = result.allNS()

	return result, nil
}

func queryNSServer(server, domain string) ([]string, bool, error) {
	domain = dns.Fqdn(domain)
	//server = strings.TrimSuffix(server, ".")
	//v("dns query: @%s NS %s", server, domain)
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeNS)
	m.RecursionDesired = false

	var in *dns.Msg
	var err error
	for i := 0; i < dnsRetry; i++ {
		in, _, err = dnsClient.Exchange(m, net.JoinHostPort(server, "53"))
		if err == nil {
			break
		} else {
			v("queryNSServer(%q, @%q) try %d, error: %s", domain, server, i+1, err)
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, false, err
	}

	v("dns query (@%s NS %s) Authoritative: %t Answer:%d NS:%d", server, domain, in.Authoritative, len(in.Answer), len(in.Ns))

	out := make([]string, 0, 2)
	for _, r := range append(in.Answer, in.Ns...) {
		if t, ok := r.(*dns.NS); ok {
			//v("dns answer NS @%s\t%s:\t%s\n", server, domain, t.Ns)
			t.Ns = cleanDomain(t.Ns)
			out = append(out, t.Ns)
		}
	}

	sort.Strings(out)
	return out, in.Authoritative, nil
}

func cleanDomain(s string) string {
	return strings.ToLower(strings.TrimSuffix(s, "."))
}
