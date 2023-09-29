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

package sllogformatprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterASCII(t *testing.T) {
	str := FilterASCII("Hello\002, \a世界\r")

	assert.Equal(t, `Hello\002, \a\r`, str, "failed to filter ASCII correctly")
}
