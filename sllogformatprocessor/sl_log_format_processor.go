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
	"fmt"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

// batch_processor is a component that accepts spans and metrics, places them
// into batches and sends downstream.
//
// batch_processor implements consumer.Traces and consumer.Metrics
//
// Batches are sent out with any of the following conditions:
// - batch size reaches cfg.SendBatchSize
// - cfg.Timeout is elapsed since the timestamp when the previous batch was sent out.
type slLogFormatProcessor struct {
	logger           *zap.Logger
	exportCtx        context.Context
	timer            *time.Timer
	timeout          time.Duration
	sendBatchSize    int
	sendBatchMaxSize int

	newItem chan any
	batch   batch

	shutdownC  chan struct{}
	goroutines sync.WaitGroup

	telemetry *slLogFormatProcessorTelemetry
}

type batch interface {
	// export the current batch
	export(ctx context.Context, sendBatchMaxSize int, returnBytes bool) (sentBatchSize int, sentBatchBytes int, err error)

	// itemCount returns the size of the current batch
	itemCount() int

	// add item to the current batch
	add(item any)
}

var _ consumer.Traces = (*slLogFormatProcessor)(nil)
var _ consumer.Metrics = (*slLogFormatProcessor)(nil)
var _ consumer.Logs = (*slLogFormatProcessor)(nil)

func newSlLogFormatProcessor(set processor.Settings, cfg *Config, batch batch, useOtel bool) (*slLogFormatProcessor, error) {
	bpt, err := newSlLogFormatProcessorTelemetry(set, useOtel)
	if err != nil {
		return nil, fmt.Errorf("error to create batch processor telemetry %w", err)
	}

	return &slLogFormatProcessor{
		logger:    set.Logger,
		exportCtx: bpt.exportCtx,
		telemetry: bpt,

		sendBatchSize:    int(cfg.SendBatchSize),
		sendBatchMaxSize: int(cfg.SendBatchMaxSize),
		timeout:          cfg.Timeout,
		newItem:          make(chan any, runtime.NumCPU()),
		batch:            batch,
		shutdownC:        make(chan struct{}, 1),
	}, nil
}

func (bp *slLogFormatProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// Start is invoked during service startup.
func (bp *slLogFormatProcessor) Start(context.Context, component.Host) error {
	bp.goroutines.Add(1)
	go bp.startProcessingCycle()
	return nil
}

// Shutdown is invoked during service shutdown.
func (bp *slLogFormatProcessor) Shutdown(context.Context) error {
	close(bp.shutdownC)

	// Wait until all goroutines are done.
	bp.goroutines.Wait()
	return nil
}

func (bp *slLogFormatProcessor) startProcessingCycle() {
	defer bp.goroutines.Done()
	bp.timer = time.NewTimer(bp.timeout)
	for {
		select {
		case <-bp.shutdownC:
		DONE:
			for {
				select {
				case item := <-bp.newItem:
					bp.processItem(item)
				default:
					break DONE
				}
			}
			// This is the close of the channel
			if bp.batch.itemCount() > 0 {
				// TODO: Set a timeout on sendTraces or
				// make it cancellable using the context that Shutdown gets as a parameter
				bp.sendItems(triggerTimeout)
			}
			return
		case item := <-bp.newItem:
			if item == nil {
				continue
			}
			bp.processItem(item)
		case <-bp.timer.C:
			if bp.batch.itemCount() > 0 {
				bp.sendItems(triggerTimeout)
			}
			bp.resetTimer()
		}
	}
}

func (bp *slLogFormatProcessor) processItem(item any) {
	bp.batch.add(item)
	sent := false
	for bp.batch.itemCount() >= bp.sendBatchSize {
		sent = true
		bp.sendItems(triggerBatchSize)
	}

	if sent {
		bp.stopTimer()
		bp.resetTimer()
	}
}

func (bp *slLogFormatProcessor) stopTimer() {
	if !bp.timer.Stop() {
		<-bp.timer.C
	}
}

func (bp *slLogFormatProcessor) resetTimer() {
	bp.timer.Reset(bp.timeout)
}

func (bp *slLogFormatProcessor) sendItems(trigger trigger) {
	sent, bytes, err := bp.batch.export(bp.exportCtx, bp.sendBatchMaxSize, bp.telemetry.detailed)
	if err != nil {
		bp.logger.Warn("Sender failed", zap.Error(err))
	} else {
		bp.telemetry.record(trigger, int64(sent), int64(bytes))
	}
}

// ConsumeTraces implements TracesProcessor
func (bp *slLogFormatProcessor) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	bp.newItem <- td
	return nil
}

// ConsumeMetrics implements MetricsProcessor
func (bp *slLogFormatProcessor) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	// First thing is convert into a different internal format
	bp.newItem <- md
	return nil
}

// ConsumeLogs implements LogsProcessor
func (bp *slLogFormatProcessor) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	bp.newItem <- ld
	return nil
}

// newBatchLogsProcessor creates a new batch processor that batches logs by size or with timeout
func newBatchLogsProcessor(set processor.Settings, next consumer.Logs, cfg *Config, useOtel bool) (*slLogFormatProcessor, error) {
	return newSlLogFormatProcessor(set, cfg, newBatchLogs(set.Logger, cfg, next), useOtel)
}
