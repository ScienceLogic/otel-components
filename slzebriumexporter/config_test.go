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

package slzebriumexporter

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(confmap.New(), cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	tests := []struct {
		filename    string
		cfg         *Config
		expectedErr string
	}{
		{
			filename: "config_verbosity.yaml",
			cfg: &Config{
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
			},
		},
		{
			filename:    "invalid_verbosity_loglevel.yaml",
			expectedErr: "1 error(s) decoding:\n\n* '' has invalid keys: loglevel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.filename))
			require.NoError(t, err)
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			err = component.UnmarshalConfig(cm, cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.cfg, cfg)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "invalid queue settings",
			cfg: &Config{
				RetrySettings: exporterhelper.RetrySettings{
					Enabled:         true,
					InitialInterval: 5000000000,
					MaxInterval:     30000000000,
					MaxElapsedTime:  300000000000,
				},
				QueueSettings: exporterhelper.QueueSettings{
					Enabled:      true,
					NumConsumers: 10,
					QueueSize:    0,
				},
				HTTPClientSettings: confighttp.HTTPClientSettings{
					Endpoint:        "https://cloud.zebrium.com",
					WriteBufferSize: 524288,
					Timeout:         30000000000,
					Headers:         map[string]configopaque.String{},
				},
				Verbosity: configtelemetry.LevelDetailed,
				ZeToken:   "0000000000000000000000000000000000000000",
			},
			expectedErr: "queue settings has invalid configuration: queue size must be positive",
		},
		{
			name: "invalid endpoint",
			cfg: &Config{
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
					Endpoint:        "bad//url?",
					WriteBufferSize: 524288,
					Timeout:         30000000000,
					Headers:         map[string]configopaque.String{},
				},
				Verbosity: configtelemetry.LevelDetailed,
				ZeToken:   "0000000000000000000000000000000000000000",
			},
			expectedErr: "\"endpoint\" must be a valid URL",
		},
		{
			name: "invalid verbosity",
			cfg: &Config{
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
				Verbosity: configtelemetry.LevelNone,
				ZeToken:   "0000000000000000000000000000000000000000",
			},
			expectedErr: "verbosity level \"None\" is not supported",
		},
		{
			name: "invalid ze_token",
			cfg: &Config{
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
				ZeToken:   "z000000000000000000000000000000000000000",
			},
			expectedErr: "ze_token invalid: encoding/hex: invalid byte: U+007A 'z'",
		},
		{
			name: "config valid",
			cfg: &Config{
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
