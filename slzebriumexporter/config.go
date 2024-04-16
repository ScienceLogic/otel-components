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
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"

	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var (
	// supportedLevels in this exporter's configuration.
	// configtelemetry.LevelNone and other future values are not supported.
	supportedLevels map[configtelemetry.Level]struct{} = map[configtelemetry.Level]struct{}{
		configtelemetry.LevelBasic:    {},
		configtelemetry.LevelNormal:   {},
		configtelemetry.LevelDetailed: {},
	}
)

// Config defines configuration for slzebrium exporter.
type Config struct {
	confighttp.ClientConfig      `mapstructure:",squash"`
	exporterhelper.QueueSettings `mapstructure:"sending_queue"`
	configretry.BackOffConfig    `mapstructure:"retry_on_failure"`

	// Verbosity defines the zebrium exporter verbosity.
	Verbosity configtelemetry.Level `mapstructure:"verbosity"`

	// ZeUrl Zebrium ZAPI authentication token
	ZeToken string `mapstructure:"ze_token"`
}

var _ component.Config = (*Config)(nil)

func mapLevel(level zapcore.Level) (configtelemetry.Level, error) {
	switch level {
	case zapcore.DebugLevel:
		return configtelemetry.LevelDetailed, nil
	case zapcore.InfoLevel:
		return configtelemetry.LevelNormal, nil
	case zapcore.WarnLevel, zapcore.ErrorLevel,
		zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		// Anything above info is mapped to 'basic' level.
		return configtelemetry.LevelBasic, nil
	default:
		return configtelemetry.LevelNone, fmt.Errorf("log level %q is not supported", level)
	}
}

func validateZeToken(token string) error {
	if len(token) != 40 {
		return errors.New("must be 40 hex characters")
	}
	dst := make([]byte, hex.DecodedLen(len(token)))
	_, err := hex.Decode(dst, []byte(token))
	return err
}

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if err := cfg.QueueSettings.Validate(); err != nil {
		return fmt.Errorf("queue settings has invalid configuration: %w", err)
	}
	if _, err := url.ParseRequestURI(cfg.Endpoint); cfg.Endpoint == "" || err != nil {
		return fmt.Errorf("\"endpoint\" must be a valid URL")
	}
	if _, ok := supportedLevels[cfg.Verbosity]; !ok {
		return fmt.Errorf("verbosity level %q is not supported", cfg.Verbosity)
	}
	if err := validateZeToken(cfg.ZeToken); err != nil {
		return fmt.Errorf("ze_token invalid: %s", err.Error())
	}
	return nil
}
