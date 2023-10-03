// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sllogformatprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sllogformatprocessor"

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func FilterASCII(in string) string {
	var sb strings.Builder
	for _, r := range in {
		if utf8.RuneLen(r) > 1 {
			continue
		}
		if unicode.IsPrint(r) {
			sb.WriteRune(r)
			continue
		}
		switch r {
		case '\a':
			sb.WriteString(`\a`)
		case '\b':
			sb.WriteString(`\b`)
		case '\t':
			sb.WriteString(`\t`)
		case '\n':
			sb.WriteString(`\n`)
		case '\f':
			sb.WriteString(`\f`)
		case '\r':
			sb.WriteString(`\r`)
		case '\v':
			sb.WriteString(`\v`)
		default:
			sb.WriteString(fmt.Sprintf("\\%03o", int(r)))
		}
	}
	return sb.String()
}
