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
	"net/http"
	"os"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/plog"
)

type loggingExporter struct {
	cfg            *Config
	log            *zap.Logger
	logsMarshaler  plog.Marshaler
	client         *http.Client
	streamTokenMap map[string]string
}

func (s *loggingExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	s.log.Info("SLZebriumExporter", zap.Int("#logs", ld.LogRecordCount()))

	if s.cfg.Verbosity != configtelemetry.LevelDetailed {
		return nil
	}

	buf, err := s.logsMarshaler.MarshalLogs(ld)
	if err != nil {
		s.log.Info("SLZebriumExporter failed to marshal", zap.String("err", err.Error()))
		return err
	}
	s.log.Info(string(buf))
	return nil
}

func (s *loggingExporter) start(ctx context.Context, host component.Host) error {
	client, err := s.cfg.HTTPClientSettings.ToClient(host, component.TelemetrySettings{Logger: s.log})
	if err != nil {
		return err
	}
	s.client = client
	return nil
}

func newLoggingExporter(logger *zap.Logger, cfg *Config) *loggingExporter {
	return &loggingExporter{
		cfg:            cfg,
		log:            logger,
		logsMarshaler:  newZeLogsMarshaler(),
		streamTokenMap: make(map[string]string),
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
