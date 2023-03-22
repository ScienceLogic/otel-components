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

package slzebriumexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/slzebriumexporter"

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/slzebriumexporter/internal/otlptext"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/slzebriumexporter/internal/otlpzapi"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type loggingExporter struct {
	config           *Config
	logger           *zap.Logger
	logsMarshaler    plog.Marshaler
	metricsMarshaler pmetric.Marshaler
	tracesMarshaler  ptrace.Marshaler
}

func (s *loggingExporter) pushLogs(_ context.Context, ld plog.Logs) error {

	var err error

	s.logger.Info("ZebriumExporter",
		zap.Int("#logs", ld.LogRecordCount()))
	err = otlpzapi.SendWorkToZapi(&otlpzapi.ZapiLogInfo{
		Config: &otlpzapi.ZapiLogConfig{
			ZeLBN:   "depricated",
			ZeURL:   s.config.Endpoint,
			ZeToken: s.config.ZeToken,
		},
		Logger: s.logger,
		LD:     &ld,
	})
	if err != nil {
		return err
	}

	if s.config.Verbosity != configtelemetry.LevelDetailed {
		return nil
	}

	buf, err := s.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		return err
	}
	s.logger.Info(string(buf))
	return nil
}

func newLoggingExporter(logger *zap.Logger, cfg *Config) *loggingExporter {
	return &loggingExporter{
		config:           cfg,
		logger:           logger,
		logsMarshaler:    otlptext.NewTextLogsMarshaler(),
		metricsMarshaler: otlptext.NewTextMetricsMarshaler(),
		tracesMarshaler:  otlptext.NewTextTracesMarshaler(),
	}
}

func loggerSync(logger *zap.Logger) func(context.Context) error {
	return func(context.Context) error {
		// Currently Sync() return a different error depending on the OS.
		// Since these are not actionable ignore them.
		err := logger.Sync()
		osErr := &os.PathError{}
		if errors.As(err, &osErr) {
			wrappedErr := osErr.Unwrap()
			if knownSyncError(wrappedErr) {
				err = nil
			}
		}
		return err
	}
}
