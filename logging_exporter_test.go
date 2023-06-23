// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package slzebriumexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.sciencelogic.com/product-engineering/ze-otel/slzebriumexporter/internal/testdata"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestLoggingZebriumExporterNoErrors(t *testing.T) {
	f := NewFactory()
	lle, err := f.CreateLogsExporter(context.Background(), exportertest.NewNopCreateSettings(), f.CreateDefaultConfig())
	require.NotNil(t, lle)
	assert.NoError(t, err)

	assert.NoError(t, lle.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, lle.ConsumeLogs(context.Background(), testdata.GenerateLogs(10)))

	assert.NoError(t, lle.Shutdown(context.Background()))
}

func TestLoggingExporterErrors(t *testing.T) {
	cfg := &Config{
		RetrySettings: exporterhelper.RetrySettings{
			Enabled:         true,
			InitialInterval: 5000000000,
			MaxInterval:     30000000000,
			MaxElapsedTime:  300000000000,
		},
		QueueSettings: exporterhelper.QueueSettings{
			Enabled:      true,
			NumConsumers: 10,
			QueueSize:    5000,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint:        "https://cloud.zebrium.com",
			WriteBufferSize: 524288,
			Timeout:         30000000000,
			Headers:         map[string]configopaque.String{},
		},
		Verbosity: configtelemetry.LevelDetailed,
		ZeToken:   "0000000000000000000000000000000000000000",
	}
	le := newLoggingExporter(zaptest.NewLogger(t), cfg)
	require.NotNil(t, le)

	err := le.pushLogs(context.Background(), plog.NewLogs())
	assert.NoError(t, err)
}
