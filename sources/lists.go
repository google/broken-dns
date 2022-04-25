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

// Package sources contains tools to return, generate, or parse various sources of domains to use as inputs.
package sources

import (
	"bufio"
	"os"
	"strings"
)

// GetList returns a list of trimmed domains (one per line) from a text file at the path provided
func GetList(list string) ([]string, error) {
	out := make([]string, 0, 100)
	file, err := os.Open(list)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// remove comments
		line = strings.SplitN(line, "#", 2)[0]
		// clean input
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "" {
			out = append(out, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
