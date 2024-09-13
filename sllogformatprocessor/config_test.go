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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(confmap.New(), cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, component.UnmarshalConfig(cm, cfg))
	assert.Equal(t,
		&Config{
			SendBatchSize:    uint32(10000),
			SendBatchMaxSize: uint32(11000),
			Timeout:          time.Second * 10,
			Profiles: []ConfigProfile{
				{
					ServiceGroup: &ConfigAttribute{
						Exp: &ConfigExpression{
							Source: "lit:default",
						},
						Rename: "ze_deployment_name",
					},
					Host: &ConfigAttribute{
						Exp: &ConfigExpression{
							Source: "body:computer",
						},
						Rename: "host",
					},
					Logbasename: &ConfigAttribute{
						Exp: &ConfigExpression{
							Op: "lc",
							Exps: []*ConfigExpression{
								{
									Op: "alphanum",
									Exps: []*ConfigExpression{
										{
											Op: "rmprefix",
											Exps: []*ConfigExpression{
												{
													Source: "body:provider.name",
												},
												{
													Source: "lit:Microsoft-Windows-",
												},
											},
										},
									},
								},
							},
						},
						Rename: "logbasename",
					},
					Labels: []*ConfigAttribute{
						{
							Exp: &ConfigExpression{
								Source: "body:channel",
							},
							Rename: "win_channel",
						},
						{
							Exp: &ConfigExpression{
								Source: "body:keywords",
							},
							Rename: "win_keywords",
						},
					},
					Message: &ConfigAttribute{
						Exp: &ConfigExpression{
							Op: "or",
							Exps: []*ConfigExpression{
								{
									Source: "body:message",
								},
								{
									Source: "body:event_data",
								},
								{
									Source: "body:keywords",
								},
							},
						},
					},
					Format: "event",
				},
			},
		}, cfg)
}

func TestValidateConfig_DefaultBatchMaxSize(t *testing.T) {
	cfg := &Config{
		SendBatchSize:    100,
		SendBatchMaxSize: 0,
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidateConfig_ValidBatchSizes(t *testing.T) {
	cfg := &Config{
		SendBatchSize:    100,
		SendBatchMaxSize: 1000,
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidateConfig_InvalidBatchSize(t *testing.T) {
	cfg := &Config{
		SendBatchSize:    1000,
		SendBatchMaxSize: 100,
	}
	assert.Error(t, cfg.Validate())
}

func TestValidateConfig_ServiceGroup(t *testing.T) {
	cfg := &Config{
		SendBatchSize:    100,
		SendBatchMaxSize: 1000,
		Profiles: []ConfigProfile{
			{
				ServiceGroup: &ConfigAttribute{
					Exp: &ConfigExpression{
						Source: "bad:default",
					},
					Rename: "ze_deployment_name",
				},
			},
		},
	}
	assert.Error(t, cfg.Validate())
}

func TestMatchProfileSkipLogic(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	// Assert type assertion for *Config
	config, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("Expected *Config but got %T", cfg)
	}

	// Update the config with the profile information
	config.Profiles = []ConfigProfile{
		{
			ServiceGroup: &ConfigAttribute{
				Rename: "service_group",
				Exp: &ConfigExpression{
					Source: "lit:default-group",
				},
			},
			Host: &ConfigAttribute{
				Rename: "host",
				Exp: &ConfigExpression{
					Source: "lit:test-host",
				},
			},
			Logbasename: &ConfigAttribute{
				Rename: "logbasename",
				Exp: &ConfigExpression{
					Source: "lit:example-log",
				},
			},
			Severity: &ConfigAttribute{
				Exp: &ConfigExpression{
					Source: "lit:INFO",
				},
			},
			Message: &ConfigAttribute{
				Exp: &ConfigExpression{
					Source: "body", // Message comes from the log body
				},
			},
		},
	}

	logger := zap.NewNop()
	mockResourceLogs := plog.NewResourceLogs()
	mockScopeLogs := mockResourceLogs.ScopeLogs().AppendEmpty()

	// Create log records: one valid and one invalid
	validLogLine := "valid log line"
	invalidLogLine := "\x00\x01\x02\x03"
	createLogRecord(mockResourceLogs, mockScopeLogs, "default-group", "test-host", "example-log", validLogLine)   // Valid log
	createLogRecord(mockResourceLogs, mockScopeLogs, "default-group", "test-host", "example-log", invalidLogLine) // Invalid log

	// Iterate over the log records and pass them to MatchProfile
	for i := 0; i < mockScopeLogs.LogRecords().Len(); i++ {
		logRecord := mockScopeLogs.LogRecords().At(i)
		gen, req, err := config.MatchProfile(logger, mockResourceLogs, mockScopeLogs, logRecord)

		if i == 1 { // Second log record with unprintable characters
			assert.ErrorIs(t, err, &NoPrintablesError{}, "expected error for log record with unprintable characters")
			assert.Empty(t, gen, "gen should be empty when log record is skipped")
			assert.Empty(t, req, "req should be empty when log record is skipped")
		} else { // First log record with valid data
			assert.NoError(t, err, "expected no error for valid log record")
			assert.NotEmpty(t, gen, "gen should not be empty for valid log record")
			assert.NotEmpty(t, req, "req should not be empty for valid log record")
		}
	}
}

// Creates a log record with specific attributes and log line
func createLogRecord(resourceLogs plog.ResourceLogs, scopeLogs plog.ScopeLogs, serviceGroup, host, logbasename, logLine string) {
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	resourceLogs.Resource().Attributes().PutStr("service_group", serviceGroup)
	resourceLogs.Resource().Attributes().PutStr("host", host)
	resourceLogs.Resource().Attributes().PutStr("logbasename", logbasename)
	logRecord.Body().SetStr(logLine)
}
