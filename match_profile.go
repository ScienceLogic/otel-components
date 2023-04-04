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

package sllogformatprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/sllogformatprocessor"

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type StreamTokenReq struct {
	Stream             string            `json:"stream"`
	Logbasename        string            `json:"logbasename"`
	ContainerLog       bool              `json:"container_log"`
	LogType            string            `json:"log_type"`
	ForwardedLog       bool              `json:"forwarded_log"`
	Tz                 string            `json:"tz"`
	ZeLogCollectorVers string            `json:"Ze_log_collector_vers"`
	Ids                map[string]string `json:"ids"`
	Cfgs               map[string]string `json:"cfgs"`
	Tags               map[string]string `json:"tags"`
}

func newStreamTokenReq() StreamTokenReq {
	return StreamTokenReq{
		Stream:             "native",
		LogType:            "otel",
		ForwardedLog:       false,
		Tz:                 time.Now().Location().String(),
		ZeLogCollectorVers: "0.1.0-otelcollector",
		Ids:                make(map[string]string),
		Cfgs:               make(map[string]string),
		Tags:               make(map[string]string),
	}
}

var severityMap map[plog.SeverityNumber]string = map[plog.SeverityNumber]string{
	plog.SeverityNumberUnspecified: "",
	plog.SeverityNumberTrace:       "TRACE",
	plog.SeverityNumberTrace2:      "TRACE",
	plog.SeverityNumberTrace3:      "TRACE",
	plog.SeverityNumberTrace4:      "TRACE",
	plog.SeverityNumberDebug:       "DEBUG",
	plog.SeverityNumberDebug2:      "DEBUG",
	plog.SeverityNumberDebug3:      "DEBUG",
	plog.SeverityNumberDebug4:      "DEBUG",
	plog.SeverityNumberInfo:        "INFO",
	plog.SeverityNumberInfo2:       "INFO",
	plog.SeverityNumberInfo3:       "INFO",
	plog.SeverityNumberInfo4:       "INFO",
	plog.SeverityNumberWarn:        "WARN",
	plog.SeverityNumberWarn2:       "WARN",
	plog.SeverityNumberWarn3:       "WARN",
	plog.SeverityNumberWarn4:       "WARN",
	plog.SeverityNumberError:       "ERROR",
	plog.SeverityNumberError2:      "ERROR",
	plog.SeverityNumberError3:      "ERROR",
	plog.SeverityNumberError4:      "ERROR",
	plog.SeverityNumberFatal:       "FATAL",
	plog.SeverityNumberFatal2:      "FATAL",
	plog.SeverityNumberFatal3:      "FATAL",
	plog.SeverityNumberFatal4:      "FATAL",
}

func evalValue(component string, in pcommon.Value) string {
	var ret string
	for _, entry := range strings.Split(component, "|") {
		keys := strings.Split(entry, ".")
		count := 0
		val := in
		for _, key := range keys {
			count++
			switch val.Type() {
			case pcommon.ValueTypeMap:
				var ok bool
				val, ok = val.Map().Get(key)
				if !ok {
					return ""
				}
			default:
				break
			}
		}
		if count < len(keys) {
			return ""
		}
		switch val.Type() {
		case pcommon.ValueTypeMap:
			return ""
		case pcommon.ValueTypeSlice:
			for idx := 0; idx < val.Slice().Len(); idx++ {
				val2 := val.Slice().At(idx)
				if ret != "" {
					ret += " "
				}
				ret += val2.AsString()
			}
		default:
			ret = val.AsString()
		}
		if ret != "" {
			break
		}
	}
	ret = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		} else if r == '\n' {
			return ' '
		}
		return -1
	}, ret)
	return ret
}

func evalElem(elem string, req *StreamTokenReq, rattr, attr pcommon.Map, body pcommon.Value, isId, setLogType bool) string {
	var ret string
	arr := strings.Split(elem, ":")
	switch arr[0] {
	case CfgSourceLit:
		ret = arr[1]
	case CfgSourceRattr:
		val, ok := rattr.Get(arr[1])
		if ok {
			ret = val.AsString()
		}
	case CfgSourceAttr:
		val, ok := attr.Get(arr[1])
		if ok {
			ret = val.AsString()
		}
	case CfgSourceBody:
		val := body
		if len(arr) > 1 {
			if val.Type() != pcommon.ValueTypeStr {
				ret = evalValue(arr[1], val)
			}
		} else {
			ret = val.AsString()
		}
	}
	id := arr[1]
	if len(arr) > 2 {
		// Apply destination label name, e.g. ze_deployment_name
		id = arr[2]
		if len(arr) > 3 {
			// Apply options
			for _, option := range arr[2:] {
				arr2 := strings.SplitN(option, "=", 2)
				switch arr2[0] {
				case CfgOptionRmprefix:
					if strings.HasPrefix(ret, arr2[1]) {
						ret = ret[len(arr2[1]):]
					}
				case CfgOptionAlphaNum:
					new := ""
					for _, c := range ret {
						if unicode.IsUpper(c) || unicode.IsLower(c) || unicode.IsDigit(c) {
							new += string(c)
						}
					}
					ret = new
				case CfgOptionLc:
					ret = strings.ToLower(ret)
				}
			}
		}
	}
	if isId {
		req.Ids[id] = ret
	} else {
		req.Cfgs[id] = ret
	}
	if setLogType {
		req.Logbasename = ret
	}
	return ret
}

func evalMessage(elem string, body pcommon.Value) string {
	var ret string
	arr := strings.Split(elem, ":")
	switch arr[0] {
	case CfgSourceBody:
		val := body
		if len(arr) > 1 {
			ret = evalValue(arr[1], val)
		} else {
			ret = val.AsString()
		}
	}
	return ret
}

func (c *Config) MatchProfile(log *zap.Logger, rl plog.ResourceLogs, ils plog.ScopeLogs, lr plog.LogRecord) (*ConfigProfile, *StreamTokenReq, error) {

	for _, profile := range c.Profiles {
		req := newStreamTokenReq()
		gen := ConfigProfile{}
		gen.ServiceGroup = evalElem(profile.ServiceGroup, &req, rl.Resource().Attributes(), lr.Attributes(), lr.Body(), true, false)
		if gen.ServiceGroup == "" {
			continue
		}
		gen.Host = evalElem(profile.Host, &req, rl.Resource().Attributes(), lr.Attributes(), lr.Body(), true, false)
		if gen.Host == "" {
			continue
		}
		gen.Logbasename = evalElem(profile.Logbasename, &req, rl.Resource().Attributes(), lr.Attributes(), lr.Body(), true, true)
		if gen.Logbasename == "" {
			continue
		}
		for _, label := range profile.Labels {
			_ = evalElem(label, &req, rl.Resource().Attributes(), lr.Attributes(), lr.Body(), false, false)
		}
		gen.Message = evalMessage(profile.Message, lr.Body())
		if gen.Message == "" {
			continue
		}
		// FORMAT MESSAGE
		switch profile.Format {
		case CfgFormatEvent:
			var timestamp time.Time
			const RFC3339Micro = "2006-01-02T15:04:05.999999Z07:00"
			if lr.Timestamp() != 0 {
				timestamp = time.Unix(0, int64(lr.Timestamp()))
			} else {
				timestamp = time.Unix(0, int64(lr.ObservedTimestamp()))
			}
			sevText, _ := severityMap[lr.SeverityNumber()]
			gen.Message = "ze_tm=" + strconv.FormatInt(timestamp.UnixMilli(), 10) + ",msg=" + timestamp.UTC().Format(RFC3339Micro) + " " + sevText + " " + gen.Message
		case CfgFormatContainer:
			req.ContainerLog = true
		}
		gen.Format = profile.Format
		return &gen, &req, nil
	}
	return nil, nil, errors.New("No matching profile for log record")
}
