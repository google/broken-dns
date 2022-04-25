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
	"flag"
	"fmt"
	"lame-dns/cache"
	"lame-dns/jobs"
	"lame-dns/sources"
	"log"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/sync/errgroup"
)

var (
	parallel = flag.Uint("parallel", 10, "number of worker threads to use")
	verbose  = flag.Bool("verbose", false, "show verbose messages")
	useLists = flag.String("list", "", "comma-separated list of domain lists")
	nsSet    = flag.String("expected-ns", "", "comma-separated list of domains which we expect nameservers to be under, findings are logged otherwise")
)

var work *jobs.Jobs
var seen *cache.Cache[[]string]
var expectedNameServers []string

func main() {
	flag.Parse()
	if flag.NArg() == 0 && *useLists == "" {
		fmt.Fprintf(os.Stderr, "need to pass at least one name or input source to scan\n")
		flag.Usage()
		return
	}

	// parse expectedNS
	for _, ns := range strings.Split(*nsSet, ",") {
		if ns != "" {
			ns = cleanDomain(ns)
			expectedNameServers = append(expectedNameServers, ns)
		}
	}

	// can't run on 0 threads
	if *parallel < 1 {
		fmt.Fprintln(os.Stderr, "must enter a positive number of parallel threads")
		flag.Usage()
		return
	}
	start := time.Now()

	seen = cache.New[[]string]()
	work = jobs.Start(context.Background())

	// start workers
	work.SaveGo(saver)
	log.Printf("starting %d threads", *parallel)
	for i := uint(0); i < *parallel; i++ {
		work.Go(worker)
	}

	// add initial jobs
	var inputGroup errgroup.Group
	// add args
	inputGroup.Go(func() error {
		addNameArray(flag.Args())
		return nil
	})

	// add lists
	for _, list := range strings.Split(*useLists, ",") {
		if list != "" {
			inputGroup.Go(func() error {
				names, err := sources.GetList(list)
				if err != nil {
					return err
				}
				addNameArray(names)
				return nil
			})
		}
	}

	// wait for all adding to be done
	err := inputGroup.Wait()
	check(err)
	// wait for all processing to be done
	err = work.Wait()
	check(err)

	v("done")
	v("took: %s", time.Since(start).Round(time.Second))
}

func addNameArray(names []string) {
	//tree.addInput(names)
	w := make([]jobs.Job, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(name)
		if _, ok := dns.IsDomainName(name); ok {
			name = strings.TrimSuffix(name, ".") // remove trailing . if domain was FQDN
			// skip arpa domains
			if strings.HasSuffix(name, ".arpa") || strings.HasSuffix(name, ".in-addr-arpa") {
				continue
			}
			w = append(w, &nameWork{Name: name})
		} else {
			log.Printf("WARNING: %q is not a DNS name, skipping", name)
		}
	}
	work.Add(w...)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func v(format string, d ...interface{}) {
	if *verbose {
		log.Printf(format, d...)
	}
}

func finding(format string, d ...interface{}) {
	format2 := "[FINDING] " + format + "\n"
	fmt.Printf(format2, d...)
	v(format2, d...)
}
