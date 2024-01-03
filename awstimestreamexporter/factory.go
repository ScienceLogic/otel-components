/* This is the interaction point with the main OTC code.
 * Templated code calls NewFactory on every configured component/plugin.
 * The helper functions wrapping the exporter code ensure the proper methods
 * are invoked.
 *
 * Only metrics exporter defined.
 */

package awstimestreamexporter

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr            = "awstimestream"
	stability          = component.StabilityLevelDevelopment
	maxRecordsPerBatch = 100
)

// NewFactory creates a factory for AWS Timestream exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		TimeoutSettings:    exporterhelper.NewDefaultTimeoutSettings(),
		RetrySettings:      exporterhelper.NewDefaultRetrySettings(),
		QueueSettings:      exporterhelper.NewDefaultQueueSettings(),
		MaxRecordsPerBatch: maxRecordsPerBatch,
		CommonAttributes:   map[string]string{},
	}
}

// Define creation of metrics exporter which is invoked by the core collector code
func createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {
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
