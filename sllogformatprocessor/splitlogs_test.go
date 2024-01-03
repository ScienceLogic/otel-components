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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.sciencelogic.com/product-engineering/ze-otel/sllogformatprocessor/internal/testdata"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestSplitLogs_noop(t *testing.T) {
	td := testdata.GenerateLogs(20)
	rl := td.ResourceLogs().At(0)
	splitSize := 40
	split := splitLogs(splitSize, rl)
	assert.Equal(t, rl, split)

	i := 0
	rl.ScopeLogs().At(0).LogRecords().RemoveIf(func(_ plog.LogRecord) bool {
		i++
		return i > 5
	})
	assert.EqualValues(t, rl, split)
}

func TestSplitLogs(t *testing.T) {
	ld := testdata.GenerateLogs(20)
	logs := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	cp := plog.NewLogs()
	cpLogs := cp.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	cpLogs.EnsureCapacity(5)
	crl := cp.ResourceLogs().At(0)
	rl := ld.ResourceLogs().At(0)
	rl.Resource().CopyTo(crl.Resource())
	rl.ScopeLogs().At(0).Scope().CopyTo(crl.ScopeLogs().At(0).Scope())
	logs.At(0).CopyTo(cpLogs.AppendEmpty())
	logs.At(1).CopyTo(cpLogs.AppendEmpty())
	logs.At(2).CopyTo(cpLogs.AppendEmpty())
	logs.At(3).CopyTo(cpLogs.AppendEmpty())
	logs.At(4).CopyTo(cpLogs.AppendEmpty())

	splitSize := 5
	split := splitLogs(splitSize, rl)
	assert.Equal(t, splitSize, resourceLRC(split))
	assert.Equal(t, crl, split)
	assert.Equal(t, 15, resourceLRC(rl))
	assert.Equal(t, "test-log-int-0-0", split.ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-4", split.ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, rl)
	assert.Equal(t, 10, resourceLRC(rl))
	assert.Equal(t, "test-log-int-0-5", split.ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-9", split.ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, rl)
	assert.Equal(t, 5, resourceLRC(rl))
	assert.Equal(t, "test-log-int-0-10", split.ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-14", split.ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, rl)
	assert.Equal(t, 5, resourceLRC(rl))
	assert.Equal(t, "test-log-int-0-15", split.ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-19", split.ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}

func TestSplitLogsMultipleILL(t *testing.T) {
	td := testdata.GenerateLogs(20)
	rl := td.ResourceLogs().At(0)
	logs := rl.ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	// add second index to ILL
	rl.ScopeLogs().At(0).
		CopyTo(rl.ScopeLogs().AppendEmpty())
	logs = rl.ScopeLogs().At(1).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(1, i))
	}

	// add third index to ILL
	rl.ScopeLogs().At(0).
		CopyTo(rl.ScopeLogs().AppendEmpty())
	logs = rl.ScopeLogs().At(2).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(2, i))
	}

	splitSize := 40
	split := splitLogs(splitSize, rl)
	assert.Equal(t, splitSize, resourceLRC(split))
	assert.Equal(t, 20, resourceLRC(rl))
	assert.Equal(t, "test-log-int-0-0", split.ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-4", split.ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}

func getTestLogSeverityText(requestNum, index int) string {
	return fmt.Sprintf("test-log-int-%d-%d", requestNum, index)
}
