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
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/pdata/plog"
)

type loggingExporter struct {
	cfg            *Config
	log            *zap.Logger
	client         *http.Client
	streamTokenMap map[string]string
}

const (
	CfgFormatMessage   string = "message"
	CfgFormatContainer string = "container"
	CfgFormatEvent     string = "event"
)

var cfgFormatMap map[string]struct{} = map[string]struct{}{
	CfgFormatMessage:   {},
	CfgFormatContainer: {},
	CfgFormatEvent:     {},
}

func keysForMap(mymap map[string]struct{}) []string {
	keys := make([]string, len(mymap))
	i := 0
	for k := range mymap {
		keys[i] = k
		i++
	}
	return keys
}

func validateResourceElem(idx int, name, str string, cfgMap map[string]struct{}) error {
	arr := strings.Split(str, ":")
	if len(arr) < 1 || len(arr[0]) < 1 {
		return fmt.Errorf("resource %d missing %s", idx, name)
	}
	_, ok := cfgMap[arr[0]]
	if !ok {
		return fmt.Errorf("resource %d invalid value %s for %s, supported values %v", idx, arr[0], name, keysForMap(cfgMap))
	}
	return nil
}

func (s *loggingExporter) pushLogs(_ context.Context, ld plog.Logs) error {
	s.log.Info("SLZebriumExporter", zap.Int("#logs", ld.LogRecordCount()))

	rls := ld.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		request, err := s.getStreamTokenRequest(rl)
		if err != nil {
			s.log.Error("Failed to get metadata", zap.String("err", err.Error()))
			return err
		}

		buffer, err := s.marshalLogs(rl)
		if err != nil {
			s.log.Error("Failed to marshal log messages", zap.String("err", err.Error()))
			return err
		}

		if s.cfg.Verbosity == configtelemetry.LevelDetailed {
			s.log.Info(request)
			s.log.Info(string(buffer))
		}

		val, _ := rl.Resource().Attributes().Get("sl_format")
		format := val.AsString()
		if err := validateResourceElem(i, "sl_format", format, cfgFormatMap); err != nil {
			return err
		}
		err = s.sendLogs(request, format, buffer)
		if err != nil {
			s.log.Error("Failed to send logs", zap.String("err", err.Error()))
			return err
		}
	}

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
