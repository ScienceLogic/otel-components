package awstimestreamexporter

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for AWS Timestream Exporter
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"`
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	Database           string            `mapstructure:"database"`
	Table              string            `mapstructure:"table"`
	Region             string            `mapstructure:"region"`
	CommonAttributes   map[string]string `mapstructure:"common_attributes"`
	MaxRecordsPerBatch int               `mapstructure:"max_records_per_batch"`
}

var _ component.Config = &Config{}

func (cfg *Config) Validate() error {
	if err := cfg.QueueSettings.Validate(); err != nil {
		return fmt.Errorf("queue settings are invalid :%w", err)
	}
	//if err := cfg.RetrySettings.Validate(); err != nil {
	//	return fmt.Errorf("retry settings are invalid :%w", err)
	//}
	if cfg.Database == "" {
		return errors.New("Database must be specified")
	}
	if cfg.Table == "" {
		return errors.New("Table must be specified")
	}

	return nil
}
