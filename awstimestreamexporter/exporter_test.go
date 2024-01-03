package awstimestreamexporter

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite/types"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func getConfig() *Config {
	return &Config{
		Database: "ae-sample-db",
		Table:    "sample-table",
		Region:   "us-east-1",
	}
}

func getTestDataSet() (pmetric.Metrics, []types.Record) {
	testTimestamp := pcommon.Timestamp(time.Date(2022, 11, 10, 00, 00, 0, 0, time.UTC).UnixNano())
	nanoTestTimestamp := strconv.FormatUint(uint64(testTimestamp), 10)
	dimensions := map[string]string{"name1": "value1", "name2": "value2"}
	dimKeys := []string{}
	for key := range dimensions {
		dimKeys = append(dimKeys, key)
	}
	dimValue0 := dimensions[dimKeys[0]]
	dimValue1 := dimensions[dimKeys[1]]

	commonDimKey := "commonKey"
	commonDimValue := "commonKey"

	md := pmetric.NewMetrics()
	md.ResourceMetrics().EnsureCapacity(2)
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr(commonDimKey, commonDimValue)

	ilms := rm.ScopeMetrics()
	ilms.EnsureCapacity(2)
	ilm := ilms.AppendEmpty()

	metrics := ilm.Metrics()

	intSumMetricName := "da_metric&sum"
	intSumMetricValue := "10"
	metrics.EnsureCapacity(2)
	intSumMetric := metrics.AppendEmpty()
	intSumMetric.SetName(intSumMetricName)
	intSumMetric.SetUnit("iu")
	intSum := intSumMetric.SetEmptySum()
	intSumDataPoints := intSum.DataPoints()
	intSumDataPoint := intSumDataPoints.AppendEmpty()
	intSumDataPoint.SetIntValue(10)
	intSumDataPoint.SetTimestamp(testTimestamp)
	intSumDataPoint.Attributes().PutStr(dimKeys[0], dimValue0)

	doubleGaugeMetricName := "da%metric_guage"
	doubleGaugeMetricValue := "20.1"
	doubleGaugeMetric := metrics.AppendEmpty()
	doubleGaugeMetric.SetName(doubleGaugeMetricName)
	doubleGaugeMetric.SetUnit("du")
	doubleGauge := doubleGaugeMetric.SetEmptyGauge()
	doubleGaugeDataPoints := doubleGauge.DataPoints()
	doubleGaugeDataPoint := doubleGaugeDataPoints.AppendEmpty()
	doubleGaugeDataPoint.SetDoubleValue(20.1)
	doubleGaugeDataPoint.SetTimestamp(testTimestamp)
	doubleGaugeDataPoint.Attributes().PutStr(dimKeys[1], dimValue1)

	intSumMetricNameInRecord := "da_metric-sum_iu"
	doubleGaugeMetricNameInRecord := "da-metric_guage_du"

	records := []types.Record{
		{
			MeasureName:      &intSumMetricNameInRecord,
			MeasureValue:     &intSumMetricValue,
			MeasureValueType: types.MeasureValueTypeBigint,
			Time:             &nanoTestTimestamp,
			TimeUnit:         types.TimeUnitNanoseconds,
			Dimensions: []types.Dimension{
				{
					Name:               &dimKeys[0],
					Value:              &dimValue0,
					DimensionValueType: types.DimensionValueTypeVarchar,
				},
				{
					Name:               &commonDimKey,
					Value:              &commonDimValue,
					DimensionValueType: types.DimensionValueTypeVarchar,
				},
			},
		},
		{
			MeasureName:      &doubleGaugeMetricNameInRecord,
			MeasureValue:     &doubleGaugeMetricValue,
			MeasureValueType: types.MeasureValueTypeDouble,
			Time:             &nanoTestTimestamp,
			TimeUnit:         types.TimeUnitNanoseconds,
			Dimensions: []types.Dimension{
				{
					Name:               &dimKeys[1],
					Value:              &dimValue1,
					DimensionValueType: types.DimensionValueTypeVarchar,
				},
				{
					Name:               &commonDimKey,
					Value:              &commonDimValue,
					DimensionValueType: types.DimensionValueTypeVarchar,
				},
			},
		},
	}

	return md, records
}

/*
func getSimpleTestDataSet() (pmetric.Metrics, []types.Record) {
	testTimestamp := pcommon.Timestamp(time.Date(2022, 11, 10, 00, 00, 0, 0, time.UTC).UnixNano())
	nanoTestTimestamp := strconv.FormatUint(uint64(testTimestamp), 10)

	md := pmetric.NewMetrics()
	md.ResourceMetrics().EnsureCapacity(2)
	rm := md.ResourceMetrics().AppendEmpty()

	ilms := rm.ScopeMetrics()
	ilms.EnsureCapacity(2)
	ilm := ilms.AppendEmpty()

	metrics := ilm.Metrics()

	doubleGaugeMetricName := "test_gauge"
	doubleGaugeMetricValue := "14.8"
	doubleGaugeMetric := metrics.AppendEmpty()
	doubleGaugeMetric.SetName(doubleGaugeMetricName)
	doubleGaugeMetric.SetUnit("du")
	doubleGauge := doubleGaugeMetric.SetEmptyGauge()
	doubleGaugeDataPoints := doubleGauge.DataPoints()
	doubleGaugeDataPoint := doubleGaugeDataPoints.AppendEmpty()
	doubleGaugeDataPoint.SetDoubleValue(14.8)
	doubleGaugeDataPoint.SetTimestamp(testTimestamp)
	doubleGaugeDataPoint.Attributes().PutStr("some", "")
	doubleGaugeDataPoint.Attributes().PutStr("other", "stuff")

	k := "some"
	v := "value"

	records := []types.Record{
		{
			MeasureName:      &doubleGaugeMetricName,
			MeasureValue:     &doubleGaugeMetricValue,
			MeasureValueType: types.MeasureValueTypeDouble,
			Time:             &nanoTestTimestamp,
			TimeUnit:         types.TimeUnitNanoseconds,
			Dimensions: []types.Dimension{
				{
					Name:               &k,
					Value:              &v,
					DimensionValueType: types.DimensionValueTypeVarchar,
				},
			},
		},
	}

	return md, records
}
*/

func TestConvertMetricsToRecords(t *testing.T) {
	e := createExporter(
		context.TODO(),
		getConfig(),
		zap.NewNop(),
		func(context.Context, string, *zap.Logger) *timestreamwrite.Client {
			return nil
		},
	)
	md, records := getTestDataSet()
	assert.Equal(t, records, e.convertMetricsToRecords(md))
}

/* integration test, disable for now */
/*
func TestIntegrationTimestream(t *testing.T) {
	e := createExporter(context.TODO(), getConfig(), zap.NewNop(), newWriteSession)
	md, _ := getSimpleTestDataSet()
	assert.NoError(t, e.pushMetrics(context.TODO(), md))
}
*/
