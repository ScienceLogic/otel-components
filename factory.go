package awstimestreamexporter

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr            = "awstimestream"
	stability          = component.StabilityLevelInDevelopment
	maxRecordsPerBatch = 100
)

// NewFactory creates a factory for AWS Timestream exporter.
func NewFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithMetricsExporter(createMetricsExporter, stability),
	)
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings:   config.NewExporterSettings(config.NewComponentID(typeStr)),
		TimeoutSettings:    exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:      exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:      exporterhelper.NewDefaultQueueSettings(),
		MaxRecordsPerBatch: maxRecordsPerBatch,
		CommonAttributes:   map[string]string{},
	}
}

func createMetricsExporter(ctx context.Context, set component.ExporterCreateSettings, cfg config.Exporter) (component.MetricsExporter, error) {
	c, ok := cfg.(*Config)
	if !ok || c == nil {
		return nil, errors.New("incorrect config provided")
	}

	exp := createExporter(ctx, c, set.Logger, newWriteSession)
	cexp, err := exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		exp.pushMetrics,
		exporterhelper.WithTimeout(c.TimeoutSettings),
		exporterhelper.WithRetry(c.RetrySettings),
		exporterhelper.WithQueue(c.QueueSettings),
	)
	return cexp, err
}
