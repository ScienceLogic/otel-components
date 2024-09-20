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

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, confmap.New().Unmarshal(cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NoError(t, cm.Unmarshal(cfg))
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
