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
	//"context"
	//"fmt"
	//"math"
	//"sync"
	"testing"
	//"time"
	//"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/require"
	//"go.opentelemetry.io/collector/component/componenttest"
	//"go.opentelemetry.io/collector/config/configtelemetry"
	//"go.opentelemetry.io/collector/consumer"
	//"go.opentelemetry.io/collector/consumer/consumertest"
	//"go.opentelemetry.io/collector/internal/testdata"
	//"go.opentelemetry.io/collector/pdata/plog"
	//"go.opentelemetry.io/collector/pdata/pmetric"
	//"go.opentelemetry.io/collector/pdata/ptrace"
	//"go.opentelemetry.io/collector/processor/processortest"
)

func TestBatchProcessorSpansDelivered(t *testing.T) {
	/*
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		cfg.SendBatchSize = 128
		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg, false)
		require.NoError(t, err)
		require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

		requestCount := 1000
		spansPerRequest := 100
		sentResourceSpans := ptrace.NewTraces().ResourceSpans()
		for requestNum := 0; requestNum < requestCount; requestNum++ {
			td := testdata.GenerateTraces(spansPerRequest)
			spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
			for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
				spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
			}
			td.ResourceSpans().At(0).CopyTo(sentResourceSpans.AppendEmpty())
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}

		// Added to test logic that check for empty resources.
		td := ptrace.NewTraces()
		assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))

		require.NoError(t, batcher.Shutdown(context.Background()))

		require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
		receivedTraces := sink.AllTraces()
		spansReceivedByName := spansReceivedByName(receivedTraces)
		for requestNum := 0; requestNum < requestCount; requestNum++ {
			spans := sentResourceSpans.At(requestNum).ScopeSpans().At(0).Spans()
			for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
				require.EqualValues(t,
					spans.At(spanIndex),
					spansReceivedByName[getTestSpanName(requestNum, spanIndex)])
			}
		}
	*/
}

func TestBatchProcessorSpansDeliveredEnforceBatchSize(t *testing.T) {
	/*
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		cfg.SendBatchSize = 128
		cfg.SendBatchMaxSize = 130
		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg, false)
		require.NoError(t, err)
		require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

		requestCount := 1000
		spansPerRequest := 150
		for requestNum := 0; requestNum < requestCount; requestNum++ {
			td := testdata.GenerateTraces(spansPerRequest)
			spans := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
			for spanIndex := 0; spanIndex < spansPerRequest; spanIndex++ {
				spans.At(spanIndex).SetName(getTestSpanName(requestNum, spanIndex))
			}
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}

		// Added to test logic that check for empty resources.
		td := ptrace.NewTraces()
		require.NoError(t, batcher.ConsumeTraces(context.Background(), td))

		// wait for all spans to be reported
		for {
			if sink.SpanCount() == requestCount*spansPerRequest {
				break
			}
			<-time.After(cfg.Timeout)
		}

		require.NoError(t, batcher.Shutdown(context.Background()))

		require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
		for i := 0; i < len(sink.AllTraces())-1; i++ {
			assert.Equal(t, int(cfg.SendBatchMaxSize), sink.AllTraces()[i].SpanCount())
		}
		// the last batch has the remaining size
		assert.Equal(t, (requestCount*spansPerRequest)%int(cfg.SendBatchMaxSize), sink.AllTraces()[len(sink.AllTraces())-1].SpanCount())
	*/
}

func TestBatchProcessorSentBySize(t *testing.T) {
	//telemetryTest(t, testBatchProcessorSentBySize)
}

/*
func testBatchProcessorSentBySize(t *testing.T, tel testTelemetry, useOtel bool) {
		sizer := &ptrace.ProtoMarshaler{}
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		sendBatchSize := 20
		cfg.SendBatchSize = uint32(sendBatchSize)
		cfg.Timeout = 500 * time.Millisecond
		creationSet := tel.NewProcessorSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg, useOtel)
		require.NoError(t, err)
		require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

		requestCount := 100
		spansPerRequest := 5

		start := time.Now()
		sizeSum := 0
		for requestNum := 0; requestNum < requestCount; requestNum++ {
			td := testdata.GenerateTraces(spansPerRequest)
			sizeSum += sizer.TracesSize(td)
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}

		require.NoError(t, batcher.Shutdown(context.Background()))

		elapsed := time.Since(start)
		require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

		expectedBatchesNum := requestCount * spansPerRequest / sendBatchSize
		expectedBatchingFactor := sendBatchSize / spansPerRequest

		require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
		receivedTraces := sink.AllTraces()
		require.EqualValues(t, expectedBatchesNum, len(receivedTraces))
		for _, td := range receivedTraces {
			rss := td.ResourceSpans()
			require.Equal(t, expectedBatchingFactor, rss.Len())
			for i := 0; i < expectedBatchingFactor; i++ {
				require.Equal(t, spansPerRequest, rss.At(i).ScopeSpans().At(0).Spans().Len())
			}
		}

		tel.assertMetrics(t, expectedMetrics{
			sendCount:        float64(expectedBatchesNum),
			sendSizeSum:      float64(sink.SpanCount()),
			sendSizeBytesSum: float64(sizeSum),
			sizeTrigger:      float64(expectedBatchesNum),
		})
}
*/

func TestBatchProcessorSentBySizeWithMaxSize(t *testing.T) {
	//telemetryTest(t, testBatchProcessorSentBySizeWithMaxSize)
}

/*
func testBatchProcessorSentBySizeWithMaxSize(t *testing.T, tel testTelemetry, useOtel bool) {
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		sendBatchSize := 20
		sendBatchMaxSize := 37
		cfg.SendBatchSize = uint32(sendBatchSize)
		cfg.SendBatchMaxSize = uint32(sendBatchMaxSize)
		cfg.Timeout = 500 * time.Millisecond
		creationSet := tel.NewProcessorSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg, useOtel)
		require.NoError(t, err)
		require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

		requestCount := 1
		spansPerRequest := 500
		totalSpans := requestCount * spansPerRequest

		start := time.Now()
		for requestNum := 0; requestNum < requestCount; requestNum++ {
			td := testdata.GenerateTraces(spansPerRequest)
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}

		require.NoError(t, batcher.Shutdown(context.Background()))

		elapsed := time.Since(start)
		require.LessOrEqual(t, elapsed.Nanoseconds(), cfg.Timeout.Nanoseconds())

		// The max batch size is not a divisor of the total number of spans
		expectedBatchesNum := int(math.Ceil(float64(totalSpans) / float64(sendBatchMaxSize)))

		require.Equal(t, totalSpans, sink.SpanCount())
		receivedTraces := sink.AllTraces()
		require.EqualValues(t, expectedBatchesNum, len(receivedTraces))

		tel.assertMetrics(t, expectedMetrics{
			sendCount:      float64(expectedBatchesNum),
			sendSizeSum:    float64(sink.SpanCount()),
			sizeTrigger:    math.Floor(float64(totalSpans) / float64(sendBatchMaxSize)),
			timeoutTrigger: 1,
		})
}
*/

func TestBatchProcessorSentByTimeout(t *testing.T) {
	/*
		sink := new(consumertest.TracesSink)
		cfg := createDefaultConfig().(*Config)
		sendBatchSize := 100
		cfg.SendBatchSize = uint32(sendBatchSize)
		cfg.Timeout = 100 * time.Millisecond

		requestCount := 5
		spansPerRequest := 10
		start := time.Now()

		creationSet := processortest.NewNopSettings()
		creationSet.MetricsLevel = configtelemetry.LevelDetailed
		batcher, err := newBatchTracesProcessor(creationSet, sink, cfg, false)
		require.NoError(t, err)
		require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

		for requestNum := 0; requestNum < requestCount; requestNum++ {
			td := testdata.GenerateTraces(spansPerRequest)
			assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
		}

		// Wait for at least one batch to be sent.
		for {
			if sink.SpanCount() != 0 {
				break
			}
			<-time.After(cfg.Timeout)
		}

		elapsed := time.Since(start)
		require.LessOrEqual(t, cfg.Timeout.Nanoseconds(), elapsed.Nanoseconds())

		// This should not change the results in the sink, verified by the expectedBatchesNum
		require.NoError(t, batcher.Shutdown(context.Background()))

		expectedBatchesNum := 1
		expectedBatchingFactor := 5

		require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
		receivedTraces := sink.AllTraces()
		require.EqualValues(t, expectedBatchesNum, len(receivedTraces))
		for _, td := range receivedTraces {
			rss := td.ResourceSpans()
			require.Equal(t, expectedBatchingFactor, rss.Len())
			for i := 0; i < expectedBatchingFactor; i++ {
				require.Equal(t, spansPerRequest, rss.At(i).ScopeSpans().At(0).Spans().Len())
			}
		}
	*/
}

func TestBatchProcessorTraceSendWhenClosing(t *testing.T) {
	/*
			cfg := Config{
				Timeout:       3 * time.Second,
				SendBatchSize: 1000,
			}
			sink := new(consumertest.TracesSink)

			creationSet := processortest.NewNopSettings()
			creationSet.MetricsLevel = configtelemetry.LevelDetailed
			batcher, err := newBatchTracesProcessor(creationSet, sink, &cfg, false)
			require.NoError(t, err)
			require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

			requestCount := 10
			spansPerRequest := 10
			for requestNum := 0; requestNum < requestCount; requestNum++ {
				td := testdata.GenerateTraces(spansPerRequest)
				assert.NoError(t, batcher.ConsumeTraces(context.Background(), td))
			}

			require.NoError(t, batcher.Shutdown(context.Background()))

			require.Equal(t, requestCount*spansPerRequest, sink.SpanCount())
			require.Equal(t, 1, len(sink.AllTraces()))
		}

		func TestBatchMetricProcessor_ReceivingData(t *testing.T) {
			// Instantiate the batch processor with low config values to test data
			// gets sent through the processor.
			cfg := Config{
				Timeout:       200 * time.Millisecond,
				SendBatchSize: 50,
			}

			requestCount := 100
			metricsPerRequest := 5
			sink := new(consumertest.MetricsSink)

			creationSet := processortest.NewNopSettings()
			creationSet.MetricsLevel = configtelemetry.LevelDetailed
			batcher, err := newBatchMetricsProcessor(creationSet, sink, &cfg, false)
			require.NoError(t, err)
			require.NoError(t, batcher.Start(context.Background(), componenttest.NewNopHost()))

			sentResourceMetrics := pmetric.NewMetrics().ResourceMetrics()

			for requestNum := 0; requestNum < requestCount; requestNum++ {
				md := testdata.GenerateMetrics(metricsPerRequest)
				metrics := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
				for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
					metrics.At(metricIndex).SetName(getTestMetricName(requestNum, metricIndex))
				}
				md.ResourceMetrics().At(0).CopyTo(sentResourceMetrics.AppendEmpty())
				assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))
			}

			// Added to test case with empty resources sent.
			md := pmetric.NewMetrics()
			assert.NoError(t, batcher.ConsumeMetrics(context.Background(), md))

			require.NoError(t, batcher.Shutdown(context.Background()))

			require.Equal(t, requestCount*2*metricsPerRequest, sink.DataPointCount())
			receivedMds := sink.AllMetrics()
			metricsReceivedByName := metricsReceivedByName(receivedMds)
			for requestNum := 0; requestNum < requestCount; requestNum++ {
				metrics := sentResourceMetrics.At(requestNum).ScopeMetrics().At(0).Metrics()
				for metricIndex := 0; metricIndex < metricsPerRequest; metricIndex++ {
					require.EqualValues(t,
						metrics.At(metricIndex),
						metricsReceivedByName[getTestMetricName(requestNum, metricIndex)])
				}
			}
	*/
}

func TestBatchMetricProcessorBatchSize(t *testing.T) {
	//telemetryTest(t, testBatchMetricProcessorBatchSize)
}
