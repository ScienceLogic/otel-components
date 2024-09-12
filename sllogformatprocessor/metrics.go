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

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	scopeName = "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sllogformatprocessor"
)

var (
	typeStr                  = componentType.String()
	processorKey             = "processor"
	processorTagKey          = tag.MustNewKey(processorKey)
	statBatchSizeTriggerSend = stats.Int64("batch_size_trigger_send", "Number of times the batch was sent due to a size trigger", stats.UnitDimensionless)
	statTimeoutTriggerSend   = stats.Int64("timeout_trigger_send", "Number of times the batch was sent due to a timeout trigger", stats.UnitDimensionless)
	statBatchSendSize        = stats.Int64("batch_send_size", "Number of units in the batch", stats.UnitDimensionless)
	statBatchSendSizeBytes   = stats.Int64("batch_send_size_bytes", "Number of bytes in batch that was sent", stats.UnitBytes)
)

type trigger int

const (
	triggerTimeout trigger = iota
	triggerBatchSize
)

func init() {
	// TODO: Find a way to handle the error.
	_ = view.Register(metricViews()...)
}

// MetricViews returns the metrics views related to batching
func metricViews() []*view.View {
	processorTagKeys := []tag.Key{processorTagKey}

	countBatchSizeTriggerSendView := &view.View{
		Name:        processorhelper.BuildCustomMetricName(typeStr, statBatchSizeTriggerSend.Name()),
		Measure:     statBatchSizeTriggerSend,
		Description: statBatchSizeTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	countTimeoutTriggerSendView := &view.View{
		Name:        processorhelper.BuildCustomMetricName(typeStr, statTimeoutTriggerSend.Name()),
		Measure:     statTimeoutTriggerSend,
		Description: statTimeoutTriggerSend.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Sum(),
	}

	distributionBatchSendSizeView := &view.View{
		Name:        processorhelper.BuildCustomMetricName(typeStr, statBatchSendSize.Name()),
		Measure:     statBatchSendSize,
		Description: statBatchSendSize.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000),
	}

	distributionBatchSendSizeBytesView := &view.View{
		Name:        processorhelper.BuildCustomMetricName(typeStr, statBatchSendSizeBytes.Name()),
		Measure:     statBatchSendSizeBytes,
		Description: statBatchSendSizeBytes.Description(),
		TagKeys:     processorTagKeys,
		Aggregation: view.Distribution(10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
			100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
			1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000),
	}

	return []*view.View{
		countBatchSizeTriggerSendView,
		countTimeoutTriggerSendView,
		distributionBatchSendSizeView,
		distributionBatchSendSizeBytesView,
	}
}

type slLogFormatProcessorTelemetry struct {
	level    configtelemetry.Level
	detailed bool
	useOtel  bool

	exportCtx context.Context

	processorAttr        []attribute.KeyValue
	batchSizeTriggerSend metric.Int64Counter
	timeoutTriggerSend   metric.Int64Counter
	batchSendSize        metric.Int64Histogram
	batchSendSizeBytes   metric.Int64Histogram
}

func newSlLogFormatProcessorTelemetry(set processor.Settings, useOtel bool) (*slLogFormatProcessorTelemetry, error) {
	exportCtx, err := tag.New(context.Background(), tag.Insert(processorTagKey, set.ID.String()))
	if err != nil {
		return nil, err
	}

	bpt := &slLogFormatProcessorTelemetry{
		useOtel:       useOtel,
		processorAttr: []attribute.KeyValue{attribute.String(processorKey, set.ID.String())},
		exportCtx:     exportCtx,
		level:         set.MetricsLevel,
		detailed:      set.MetricsLevel == configtelemetry.LevelDetailed,
	}

	err = bpt.createOtelMetrics(set.MeterProvider)
	if err != nil {
		return nil, err
	}

	return bpt, nil
}

func (bpt *slLogFormatProcessorTelemetry) createOtelMetrics(mp metric.MeterProvider) error {
	return nil
}

func (bpt *slLogFormatProcessorTelemetry) record(trigger trigger, sent, bytes int64) {
}

func (bpt *slLogFormatProcessorTelemetry) recordWithOC(trigger trigger, sent, bytes int64) {
}

func (bpt *slLogFormatProcessorTelemetry) recordWithOtel(trigger trigger, sent int64, bytes int64) {
}
