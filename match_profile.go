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
	"encoding/json"
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
	plog.SeverityNumberUnspecified: "INFO",
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

type Operator func(string, string) string

var ops map[string]Operator = map[string]Operator{
	"||": func(a, b string) string {
		if len(a) > 0 {
			return a
		}
		return b
	},
	"+": func(a, b string) string { return a + " " + b },
}

func nextToken(in string) (string, string, string) {
	idx := len(in)
	op := ""
	for op2, _ := range ops {
		idx2 := strings.Index(in, op2)
		if idx2 < 0 {
			continue
		}
		if idx2 < idx {
			idx = idx2
			op = op2
		}
	}
	return in[:idx], op, in[idx+len(op):]
}

func evalValue(component string, val pcommon.Value) string {
	var ret string
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

func evalMap(elem string, in pcommon.Map) string {
	arr := strings.Split(elem, ".")
	if len(arr) < 1 {
		return ""
	}
	path := ""
	for idx, key := range arr {
		if path == "" {
			path = key
		} else {
			path += "." + key
		}
		val, ok := in.Get(path)
		if ok {
			if val.Type() == pcommon.ValueTypeMap {
				in = val.Map()
				path = ""
				continue
			}
			if len(arr) > idx+1 {
				elem = arr[idx+1]
			}
			return evalValue(elem, val)
		}
	}
	return ""
}

type Parser struct {
	Rattr pcommon.Map
	Attr  pcommon.Map
	Body  pcommon.Value
}

func (p *Parser) evalToken(elem string) (string, string) {
	var ret, replaceFrom, replaceTo string
	doReplace := strings.HasPrefix(elem, "replace(") && elem[len(elem)-1] == ')'
	if doReplace {
		replaceFrom = elem[len("replace("):]
		arr4 := strings.Split(replaceFrom, ",")
		replaceFrom = arr4[1]
		if len(arr4) > 2 {
			replaceTo = arr4[2][:len(arr4[2])-1]
		}
		elem = elem[len("replace("):strings.Index(elem, ",")]
	}
	arr3 := strings.SplitN(elem, ".", 2)
	id := ""
	if len(arr3) > 1 {
		id = arr3[1]
	}
	switch arr3[0] {
	case CfgSourceLit:
		ret = id
	case CfgSourceRattr:
		ret = evalMap(id, p.Rattr)
	case CfgSourceAttr:
		ret = evalMap(id, p.Attr)
	case CfgSourceBody:
		switch p.Body.Type() {
		case pcommon.ValueTypeMap:
			ret = evalMap(id, p.Body.Map())
		case pcommon.ValueTypeStr:
			raw := make(map[string]any)
			if id != "" && json.Unmarshal([]byte(p.Body.AsString()), &raw) == nil {
				av := pcommon.NewValueEmpty()
				if av.SetEmptyMap().FromRaw(raw) == nil {
					ret = evalMap(id, av.Map())
					break
				}
			}
			fallthrough
		default:
			ret = evalValue(id, p.Body)
		}
	}
	if doReplace {
		ret = strings.Replace(ret, replaceFrom, replaceTo, -1)
	}
	return id, ret
}

func (p *Parser) EvalElem(elem string) (string, string) {
	var id, ret, ret2, op string
	arr := strings.Split(elem, ":")
	text := arr[0]
	for len(text) > 0 {
		var token2, op2 string
		token2, op2, text = nextToken(text)
		id, ret2 = p.evalToken(token2)
		if op != "" {
			ret = (ops[op])(ret, ret2)
		} else {
			ret = ret2
		}
		op = op2
	}
	if len(arr) > 1 {
		// Apply destination label name, e.g. ze_deployment_name
		id = arr[1]
		if len(arr) > 2 {
			// Apply options
			for _, option := range arr[2:] {
				arr2 := strings.SplitN(option, "=", 2)
				switch arr2[0] {
				case CfgOptionRmprefix:
					if strings.HasPrefix(ret, arr2[1]) {
						ret = ret[len(arr2[1]):]
					}
				case CfgOptionRmtail:
					idx := strings.LastIndex(ret, arr2[1])
					if idx > -1 {
						ret = ret[:idx]
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
	return id, ret
}

func (c *Config) MatchProfile(log *zap.Logger, rl plog.ResourceLogs, ils plog.ScopeLogs, lr plog.LogRecord) (*ConfigProfile, *StreamTokenReq, error) {
	var id, ret string
	for _, profile := range c.Profiles {
		req := newStreamTokenReq()
		gen := ConfigProfile{}
		parser := Parser{
			Rattr: rl.Resource().Attributes(),
			Attr:  lr.Attributes(),
			Body:  lr.Body(),
		}
		id, gen.ServiceGroup = parser.EvalElem(profile.ServiceGroup)
		if gen.ServiceGroup == "" {
			continue
		}
		req.Ids[id] = gen.ServiceGroup
		id, gen.Host = parser.EvalElem(profile.Host)
		if gen.Host == "" {
			continue
		}
		req.Ids[id] = gen.Host
		id, gen.Logbasename = parser.EvalElem(profile.Logbasename)
		if gen.Logbasename == "" {
			continue
		}
		if profile.HttpStatus != "" {
			_, status := parser.EvalElem(profile.HttpStatus)
			if status == "" {
				continue
			}
			sevNum := plog.SeverityNumberUnspecified
			switch status[0] {
			case '1', '2':
				sevNum = plog.SeverityNumberInfo
			case '3':
				sevNum = plog.SeverityNumberDebug
			case '4', '5':
				sevNum = plog.SeverityNumberError
			}
			lr.SetSeverityNumber(sevNum)
		}
		req.Ids[id] = gen.Logbasename
		req.Logbasename = gen.Logbasename
		for _, label := range profile.Labels {
			id, ret = parser.EvalElem(label)
			req.Cfgs[id] = ret
		}
		_, gen.Message = parser.EvalElem(profile.Message)
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
			if len(gen.Message) > 2 && gen.Message[0] == '{' {
				// I use 2 above because we are inserting severity with a comma after,
				// so we expect both open & close with something inbeteen
				gen.Message = "ze_tm=" + strconv.FormatInt(timestamp.UnixMilli(), 10) + `,msg={"severity":"` + sevText + `",` + gen.Message[1:]
			} else {
				gen.Message = "ze_tm=" + strconv.FormatInt(timestamp.UnixMilli(), 10) + ",msg=" + timestamp.UTC().Format(RFC3339Micro) + " " + sevText + " " + gen.Message
			}
		case CfgFormatContainer:
			req.ContainerLog = true
		}
		gen.Format = profile.Format
		return &gen, &req, nil
	}
	return nil, nil, errors.New("No matching profile for log record")
}
