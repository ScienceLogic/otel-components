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
	"fmt"
	"regexp"
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

// A formatted container log entry
type ContainerLogEntry struct {
	Log       string `json:"log"`
	Timestamp string `json:"timestamp"`
	Stream    string `json:"stream"`
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
	plog.SeverityNumberUnspecified: "UNKNOWN",
	plog.SeverityNumberTrace:       "TRACE",
	plog.SeverityNumberTrace2:      "TRACE",
	plog.SeverityNumberTrace3:      "TRACE",
	plog.SeverityNumberTrace4:      "TRACE",
	plog.SeverityNumberDebug:       "DEBUG",
	plog.SeverityNumberDebug2:      "DEBUG",
	plog.SeverityNumberDebug3:      "DEBUG",
	plog.SeverityNumberDebug4:      "DEBUG",
	plog.SeverityNumberInfo:        "INFO",
	plog.SeverityNumberInfo2:       "NOTICE",
	plog.SeverityNumberInfo3:       "NOTICE",
	plog.SeverityNumberInfo4:       "NOTICE",
	plog.SeverityNumberWarn:        "WARN",
	plog.SeverityNumberWarn2:       "WARN",
	plog.SeverityNumberWarn3:       "WARN",
	plog.SeverityNumberWarn4:       "WARN",
	plog.SeverityNumberError:       "ERROR",
	plog.SeverityNumberError2:      "CRITICAL",
	plog.SeverityNumberError3:      "ALERT",
	plog.SeverityNumberError4:      "ALERT",
	plog.SeverityNumberFatal:       "FATAL",
	plog.SeverityNumberFatal2:      "FATAL",
	plog.SeverityNumberFatal3:      "FATAL",
	plog.SeverityNumberFatal4:      "FATAL",
}

var sevTextMap map[string]plog.SeverityNumber = map[string]plog.SeverityNumber{
	"TRACE":         plog.SeverityNumberTrace,
	"DEBUG":         plog.SeverityNumberDebug,
	"7":             plog.SeverityNumberDebug,
	"INFORMATIONAL": plog.SeverityNumberInfo,
	"INFO":          plog.SeverityNumberInfo,
	"NORMAL":        plog.SeverityNumberInfo,
	"6":             plog.SeverityNumberInfo,
	"NOTICE":        plog.SeverityNumberInfo2,
	"5":             plog.SeverityNumberInfo2,
	"WARNING":       plog.SeverityNumberWarn,
	"WARN":          plog.SeverityNumberWarn,
	"4":             plog.SeverityNumberWarn,
	"ERROR":         plog.SeverityNumberError,
	"ERR":           plog.SeverityNumberError,
	"3":             plog.SeverityNumberError,
	"CRITICAL":      plog.SeverityNumberError2,
	"CRIT":          plog.SeverityNumberError2,
	"2":             plog.SeverityNumberError2,
	"ALERT":         plog.SeverityNumberError3,
	"1":             plog.SeverityNumberError3,
	"FATAL":         plog.SeverityNumberFatal,
	"EMERGENCY":     plog.SeverityNumberFatal,
	"EMERG":         plog.SeverityNumberFatal,
	"PANIC":         plog.SeverityNumberFatal,
	"0":             plog.SeverityNumberFatal,
}

var sevText2Num map[string]plog.SeverityNumber = map[string]plog.SeverityNumber{
	"Unspecified": plog.SeverityNumberUnspecified,
	"Trace":       plog.SeverityNumberTrace,
	"Trace2":      plog.SeverityNumberTrace2,
	"Trace3":      plog.SeverityNumberTrace3,
	"Trace4":      plog.SeverityNumberTrace4,
	"Debug":       plog.SeverityNumberDebug,
	"Debug2":      plog.SeverityNumberDebug2,
	"Debug3":      plog.SeverityNumberDebug3,
	"Debug4":      plog.SeverityNumberDebug4,
	"Info":        plog.SeverityNumberInfo,
	"Information": plog.SeverityNumberInfo,
	"Info2":       plog.SeverityNumberInfo2,
	"Info3":       plog.SeverityNumberInfo3,
	"Info4":       plog.SeverityNumberInfo4,
	"Warn":        plog.SeverityNumberWarn,
	"Warn2":       plog.SeverityNumberWarn2,
	"Warn3":       plog.SeverityNumberWarn3,
	"Warn4":       plog.SeverityNumberWarn4,
	"Error":       plog.SeverityNumberError,
	"Error2":      plog.SeverityNumberError2,
	"Error3":      plog.SeverityNumberError3,
	"Error4":      plog.SeverityNumberError4,
	"Fatal":       plog.SeverityNumberFatal,
	"Fatal2":      plog.SeverityNumberFatal2,
	"Fatal3":      plog.SeverityNumberFatal3,
	"Fatal4":      plog.SeverityNumberFatal4,
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
	for op2 := range ops {
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
		return val.AsString()
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
	var val pcommon.Value
	var ok bool
	for idx, key := range arr {
		if path == "" {
			path = key
		} else {
			path += "." + key
		}
		val, ok = in.Get(path)
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
	if path == "" {
		return val.AsString()
	}
	return ""
}

type Parser struct {
	Log   *zap.Logger
	Rattr pcommon.Map
	Attr  pcommon.Map
	Body  pcommon.Value
}

func (p *Parser) evalExp(exp *ConfigExpression) (string, string) {
	if exp == nil {
		return "", ""
	}
	var id, ret string
	if exp.Source != "" {
		arr := strings.SplitN(exp.Source, ":", 2)
		if len(arr) > 1 {
			id = arr[1]
		}
		switch arr[0] {
		case CfgSourceLit:
			ret = id
		case CfgSourceRattr:
			ret = evalMap(id, p.Rattr)
		case CfgSourceAttr:
			ret = evalMap(id, p.Attr)
		case CfgSourceBody:
			switch p.Body.Type() {
			case pcommon.ValueTypeMap:
				if id != "" {
					ret = evalMap(id, p.Body.Map())
				} else {
					ret = p.Body.AsString()
				}
			case pcommon.ValueTypeStr:
				raw := make(map[string]any)
				if id != "" {
					err := json.Unmarshal([]byte(p.Body.AsString()), &raw)
					if err != nil {
						// Can't index into non object
						return "", ""
					}
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
		ret = FilterASCII(ret)
	} else { // Op must be populated
		var ret2 string
		id, ret = p.evalExp(exp.Exps[0])
		numExps, _ := cfgOpMap[exp.Op]
		if numExps == CMaxNumExps {
			for _, exp2 := range exp.Exps[1:] {
				_, ret2 = p.evalExp(exp2)
				switch exp.Op {
				case CfgOpAnd:
					ret += ` ` + ret2
				case CfgOpOr:
					if ret == "" {
						ret = ret2
					}
				}
			}
		}
		if numExps > 1 {
			_, ret2 = p.evalExp(exp.Exps[1])
		}
		switch exp.Op {
		case CfgOpRmprefix:
			if strings.HasPrefix(ret, ret2) {
				ret = ret[len(ret2):]
			}
		case CfgOpRmsuffix:
			if strings.HasSuffix(ret, ret2) {
				ret = ret[:len(ret)-len(ret2)]
			}
		case CfgOpRmtail:
			idx := strings.LastIndex(ret, ret2)
			if idx > -1 {
				ret = ret[:idx]
			}
		case CfgOpAlphaNum:
			var new string
			for _, c := range ret {
				if unicode.IsUpper(c) || unicode.IsLower(c) || unicode.IsDigit(c) {
					new += string(c)
				}
			}
			ret = new
		case CfgOpLc:
			ret = strings.ToLower(ret)
		case CfgOpUnescape:
			// Remove the ESC character. This is special-cased because
			// embedding escapes into configurations/configs can be
			// problematic.
			ret = strings.Replace(ret, "\x1B", "", -1)
		case CfgOpReplace:
			_, ret3 := p.evalExp(exp.Exps[2])
			ret = strings.Replace(ret, ret2, ret3, -1)
		case CfgOpRegexp:
			r, err := regexp.Compile(ret2)
			if err == nil {
				arr := r.FindStringSubmatch(ret)
				ret = ""
				if len(arr) > 1 {
					ret = strings.Join(arr[1:], "")
				}
			} else {
				p.Log.Info("failed to compile regexp",
					zap.String("id", id),
					zap.String("value", ret2))
			}
		}
	}
	return id, ret
}

func (p *Parser) EvalElem(attribute *ConfigAttribute) (string, string) {
	if attribute == nil {
		return "", ""
	}
	id, ret := p.evalExp(attribute.Exp)
	if attribute.Rename != "" {
		id = attribute.Rename
	}
	if attribute.Validate != "" {
		r := regexp.MustCompile(attribute.Validate)
		if !r.MatchString(ret) {
			p.Log.Info("failed to validate regexp",
				zap.String("id", id),
				zap.String("regexp", attribute.Validate),
				zap.String("value", ret))
			ret = ""
		}
	}
	return id, ret
}

type ConfigResult struct {
	ServiceGroup string   `mapstructure:"service_group"`
	Host         string   `mapstructure:"host"`
	Logbasename  string   `mapstructure:"logbasename"`
	Severity     string   `mapstructure:"severity"`
	Labels       []string `mapstructure:"labels"`
	Message      string   `mapstructure:"message"`
	Format       string   `mapstructure:"format"`
}

func (c *Config) MatchProfile(log *zap.Logger, rl plog.ResourceLogs, ils plog.ScopeLogs, lr plog.LogRecord) (*ConfigResult, *StreamTokenReq, error) {
	var id, ret string
	reasons := []string{}
	for _, profile := range c.Profiles {
		req := newStreamTokenReq()
		gen := ConfigResult{}
		parser := Parser{
			Log:   log,
			Rattr: rl.Resource().Attributes(),
			Attr:  lr.Attributes(),
			Body:  lr.Body(),
		}
		id, gen.ServiceGroup = parser.EvalElem(profile.ServiceGroup)
		if gen.ServiceGroup == "" {
			reasons = append(reasons, "service_group")
			continue
		}
		req.Ids[id] = gen.ServiceGroup
		id, gen.Host = parser.EvalElem(profile.Host)
		if gen.Host == "" {
			reasons = append(reasons, "host")
			continue
		}
		req.Ids[id] = gen.Host
		id, gen.Logbasename = parser.EvalElem(profile.Logbasename)
		if gen.Logbasename == "" {
			reasons = append(reasons, "logbasename")
			continue
		}
		if lr.SeverityNumber() == plog.SeverityNumberUnspecified {
			sevNum, ok := sevText2Num[lr.SeverityText()]
			if ok {
				lr.SetSeverityNumber(sevNum)
			}
		}
		if profile.Severity != nil {
			_, sevText := parser.EvalElem(profile.Severity)
			if sevText == "" {
				reasons = append(reasons, "severity")
				continue
			}
			sevText = strings.ToUpper(sevText)
			sevNum := plog.SeverityNumberUnspecified
			sevNum, _ = sevTextMap[sevText]
			if sevNum == plog.SeverityNumberUnspecified &&
				len(sevText) == 3 {
				// Interpret as HTTP status
				switch sevText[0] {
				case '1', '2':
					sevNum = plog.SeverityNumberInfo
				case '3':
					sevNum = plog.SeverityNumberDebug
				case '4', '5':
					sevNum = plog.SeverityNumberError
				}
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
			reasons = append(reasons, "message")
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
			if len(gen.Message) > 2 && gen.Message[0] == '{' {
				var contLog ContainerLogEntry
				err := json.Unmarshal([]byte(gen.Message), &contLog)
				if err == nil {
					gen.Message = contLog.Timestamp + " " + contLog.Log
				}
			}
		}
		fmt.Printf("MSG bytes: %+v\n", []byte(gen.Message))
		fmt.Printf("MSG string: %s\n", gen.Message)
		gen.Format = profile.Format
		return &gen, &req, nil
	}
	return nil, nil, fmt.Errorf("No matching profile for log record, failed to find %v", reasons)
}
