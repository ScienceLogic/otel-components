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
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

type batchLogs struct {
	log          *zap.Logger
	cfg          *Config
	nextConsumer consumer.Logs
	logData      map[string]plog.ResourceLogs
	logCount     int
	sizer        plog.Sizer
}

func newBatchLogs(log *zap.Logger, cfg *Config, nextConsumer consumer.Logs) *batchLogs {
	return &batchLogs{
		log:          log,
		cfg:          cfg,
		nextConsumer: nextConsumer,
		logData:      make(map[string]plog.ResourceLogs),
		sizer:        &plog.ProtoMarshaler{},
	}
}

func (bl *batchLogs) export(ctx context.Context, sendBatchMaxSize int, returnBytes bool) (int, int, error) {
	var req plog.Logs
	var sent int
	var bytes int
	req = plog.NewLogs()
	bl.logCount = 0
	for key, rl := range bl.logData {
		remove := true
		newCount := resourceLRC(rl)
		if newCount > 0 {
			var newRl plog.ResourceLogs
			if sendBatchMaxSize > 0 && newCount > sendBatchMaxSize {
				newRl = splitLogs(sendBatchMaxSize, rl)
				// recalculate maximum size
				newCount -= sendBatchMaxSize
				if newCount > bl.logCount {
					bl.logCount = newCount
				}
				remove = false
			} else {
				newRl = rl
			}
			newRl.MoveTo(req.ResourceLogs().AppendEmpty())
		}
		if remove {
			delete(bl.logData, key)
		}
	}
	sent = req.LogRecordCount()
	if returnBytes {
		bytes = bl.sizer.LogsSize(req)
	}
	return sent, bytes, bl.nextConsumer.ConsumeLogs(ctx, req)
}

func (bl *batchLogs) itemCount() int {
	return bl.logCount
}

func (bl *batchLogs) add(item any) {
	ld := item.(plog.Logs)

	newLogsCount := ld.LogRecordCount()
	if newLogsCount == 0 {
		return
	}
	bl.addToBatch(ld)
}

func (bl *batchLogs) addToBatch(ld plog.Logs) {

	ld.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		rl.ScopeLogs().RemoveIf(func(ils plog.ScopeLogs) bool {
			ils.LogRecords().RemoveIf(func(lr plog.LogRecord) bool {
				gen, req, err := bl.cfg.MatchProfile(bl.log, rl, ils, lr)
				if err != nil {
					bl.log.Error("Failed to match profile",
						zap.String("err", err.Error()))
					return true
				}
				keyBytes, err := json.Marshal(req)
				if err != nil {
					bl.log.Error("Field to marshal metadata",
						zap.String("err", err.Error()))
					return true
				}
				key := string(keyBytes)
				dest, ok := bl.logData[key]
				if !ok {
					dest = plog.NewResourceLogs()
					dest.Resource().Attributes().PutStr("sl_service_group", gen.ServiceGroup)
					dest.Resource().Attributes().PutStr("sl_host", gen.Host)
					dest.Resource().Attributes().PutStr("sl_logbasename", req.Logbasename)
					dest.Resource().Attributes().PutStr("sl_format", gen.Format)
					dest.Resource().Attributes().PutStr("sl_metadata", key)
					bl.logData[key] = dest
				}
				lr.Attributes().PutStr("sl_msg", gen.Message)
				if dest.ScopeLogs().Len() < 1 {
					_ = dest.ScopeLogs().AppendEmpty()
				}
				destIls := dest.ScopeLogs().At(0)
				lr.MoveTo(destIls.LogRecords().AppendEmpty())
				if destIls.LogRecords().Len() > bl.logCount {
					bl.logCount = destIls.LogRecords().Len()
				}
				return true
			})
			return true
		})
		return true
	})
}
