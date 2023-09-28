package awstimestreamexporter

import (
	"context"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite/types"
	"github.com/aws/smithy-go"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"regexp"
	"strconv"
	"strings"
	"reflect"
	"errors"
)

type timestreamExporter struct {
	writeSession *timestreamwrite.Client
	logger       *zap.Logger
	database     string
	table        string
}

var tsIllegals *regexp.Regexp = regexp.MustCompile(`[^\w\+\.=:/ ]`)

const tsReplace = "-"

// Removes characters that are illegal in Timestream records and replaces
// them with a legal character.
func removeIllegalChars(in string) string {
	return tsIllegals.ReplaceAllString(in, tsReplace)
}

// Converts OTEL attributes to timestream dimensions.  Accepts
// multiple grouping of attributes as parameters.
func convertAttrsToDimensions(attrsBundle ...*pcommon.Map) []types.Dimension {
	dimensions := []types.Dimension{}
	toDimension := func(name string, value pcommon.Value) bool {
		legalName := removeIllegalChars(name)
		legalValue := removeIllegalChars(value.AsString())
		if legalName != "" && legalValue != "" {
			dimensions = append(dimensions, types.Dimension{
				Name:               &legalName,
				Value:              &legalValue,
				DimensionValueType: types.DimensionValueTypeVarchar,
			})
		}
		return true
	}
	for _, attrs := range attrsBundle {
		attrs.Range(toDimension)
	}

	return dimensions
}

// Converts OTEL data to timestream records.
func (e *timestreamExporter) convertMetricsToRecords(md pmetric.Metrics) []types.Record {
	records := []types.Record{}
	resourceMetrics := md.ResourceMetrics()

	// Resources Level
	for i := 0; i < resourceMetrics.Len(); i++ {
		resourceMetric := resourceMetrics.At(i)
		resourceAttrs := resourceMetric.Resource().Attributes()
		scopeMetrics := resourceMetric.ScopeMetrics()

		// Scopes Level
		for j := 0; j < scopeMetrics.Len(); j++ {
			scopeMetric := scopeMetrics.At(j)
			scopeAttrs := scopeMetric.Scope().Attributes()
			metrics := scopeMetric.Metrics()

			// Metrics Level
			for k := 0; k < metrics.Len(); k++ {
				metric := metrics.At(k)
				// sanitize metric name
				measureName := removeIllegalChars(strings.Join([]string{metric.Name(), metric.Unit()}, "_"))

				var dps pmetric.NumberDataPointSlice
				if (metric.Type() == pmetric.MetricTypeGauge) || (metric.Type() == pmetric.MetricTypeSum && metric.Sum().IsMonotonic()) {
					dps = metric.Gauge().DataPoints()
				} else if metric.Type() == pmetric.MetricTypeSum {
					dps = metric.Sum().DataPoints()
				} else {
					e.logger.Error("Invalid metric type",
						zap.String("Metric Type", metric.Type().String()),
					)
					continue
				}

				// Datapoints Level
				for l := 0; l < dps.Len(); l++ {
					dp := dps.At(l)
					var measureValue string
					var measureValueType types.MeasureValueType

					// Convert measurements values to string and set proper numeric types
					if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
						measureValue = strconv.FormatInt(dp.IntValue(), 10)
						measureValueType = types.MeasureValueTypeBigint
					} else if dp.ValueType() == pmetric.NumberDataPointValueTypeDouble {
						measureValue = strconv.FormatFloat(dp.DoubleValue(), 'f', -1, 64)
						measureValueType = types.MeasureValueTypeDouble
					} else {
						e.logger.Error("Invalid measurement value type",
							zap.String("Measurement Value Type", string(dp.ValueType())),
						)
						continue
					}
					measureTime := strconv.FormatUint(uint64(dp.Timestamp()), 10)
					attributes := dp.Attributes()

					// Attributes from resource, scope, and this datapoints are put into the
					// dimensions for this timestream record.
					record := types.Record{
						MeasureName:      &measureName,
						MeasureValue:     &measureValue,
						MeasureValueType: measureValueType,
						Time:             &measureTime,
						TimeUnit:         types.TimeUnitNanoseconds,
						Dimensions:       convertAttrsToDimensions(&attributes, &resourceAttrs, &scopeAttrs),
					}
					records = append(records, record)

					e.logger.Debug("New record added", zap.Any("Record", record))
				}
			}
		}
	}

	return records
}


// main entrypoint for metrics exporting
func (e *timestreamExporter) pushMetrics(ctx context.Context, md pmetric.Metrics) error {
	e.logger.Info("Starting push metrics...")
	records := e.convertMetricsToRecords(md)
	// TODO: Possibly batch these records when I require out how the backend
	// works better.  Some of the exporter helpers may handle this already.
	writeRecordsInput := &timestreamwrite.WriteRecordsInput{
		DatabaseName: &e.database,
		TableName:    &e.table,
		Records:      records,
	}

	writeOut, err := e.writeSession.WriteRecords(ctx, writeRecordsInput)
	e.logger.Info("Timestream Write Status", zap.Any("Write Status", writeOut))

	var oe *smithy.OperationError
	if err != nil {
		e.logger.Error("Write records failed", zap.String("Error", err.Error()))
		e.logger.Error("Error:", zap.Any("error", err))
		e.logger.Error("Type of Error:", zap.String("Error type", reflect.TypeOf(err).String()))
		if errors.As(err, &oe) {
			e.logger.Error("Failed to call service:", zap.String("service", oe.Service()), zap.String("operation", oe.Operation()), zap.Error(oe.Unwrap()))
		}else {
			// TODO:  Below is not working!!!!???????
			if reject, ok := err.(*types.RejectedRecordsException); ok {
				e.logger.Error("Reject", zap.Any("reject", reject))
				e.logger.Error(
					"Records Rejected",
					zap.String("ErrorCode", reject.ErrorCode()),
					zap.String("ErrorMessage", reject.ErrorMessage()),
				)
			}
		}
		e.logger.Debug("Records:", zap.Any("records", records))
		return err
	} else {
		e.logger.Info("Write records is successful")
	}

	e.logger.Info("Done push")

	return nil
}

type sessionCreator func(ctx context.Context, region string, log *zap.Logger) *timestreamwrite.Client

func createExporter(ctx context.Context, conf *Config, log *zap.Logger, s sessionCreator) *timestreamExporter {
	return &timestreamExporter{
		writeSession: s(ctx, conf.Region, log),
		database:     conf.Database,
		table:        conf.Table,
		logger:       log,
	}
}

// create Timestream writer session with proper config
func newWriteSession(ctx context.Context, region string, log *zap.Logger) *timestreamwrite.Client {
	/*
		tr := &http.Transport{
			ResponseHeaderTimeout: 20 * time.Second,
			// Using DefaultTransport values for other parameters: https://golang.org/pkg/net/http/#RoundTripper
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				KeepAlive: 30 * time.Second,
				DualStack: true,
				Timeout:   30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		http2.ConfigureTransport(tr)
	*/
	cfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		//awsconfig.WithHTTPClient(http.Client{Transport: tr}),
	)
	if err != nil {
		log.Fatal("failed to load configuration", zap.String("Error", err.Error()))
	}

	return timestreamwrite.NewFromConfig(cfg)
}
