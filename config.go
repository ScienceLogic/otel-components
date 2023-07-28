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
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config defines configuration for batch processor.
type Config struct {
	// Science Logic input profiles
	Profiles []ConfigProfile `mapstructure:"profiles"`

	// Timeout sets the time after which a batch will be sent regardless of size.
	Timeout time.Duration `mapstructure:"timeout"`

	// SendBatchSize is the size of a batch which after hit, will trigger it to be sent.
	SendBatchSize uint32 `mapstructure:"send_batch_size"`

	// SendBatchMaxSize is the maximum size of a batch. It must be larger than SendBatchSize.
	// Larger batches are split into smaller units.
	// Default value is 0, that means no maximum size.
	SendBatchMaxSize uint32 `mapstructure:"send_batch_max_size"`
}

var _ component.Config = (*Config)(nil)

const (
	CfgSourceRattr     string = "rattr"
	CfgSourceAttr      string = "attr"
	CfgSourceBody      string = "body"
	CfgSourceLit       string = "lit"
	CfgFormatMessage   string = "message"
	CfgFormatContainer string = "container"
	CfgFormatEvent     string = "event"
	CfgOpRmprefix      string = "rmprefix"
	CfgOpRmsuffix      string = "rmsuffix"
	CfgOpRmtail        string = "rmtail"
	CfgOpAlphaNum      string = "alphanum"
	CfgOpLc            string = "lc"
	CfgOpReplace       string = "replace"
	CfgOpRegexp        string = "regexp"
	CfgOpAnd           string = "and"
	CfgOpOr            string = "or"
)

var cfgIdNames map[string]int = map[string]int{
	"service_group": 0,
	"host":          0,
	"logbasename":   0,
}

var cfgSourceMap map[string]int = map[string]int{
	CfgSourceRattr: 0,
	CfgSourceAttr:  0,
	CfgSourceBody:  0,
	CfgSourceLit:   0,
}

var cfgFormatMap map[string]int = map[string]int{
	CfgFormatMessage:   0,
	CfgFormatContainer: 0,
	CfgFormatEvent:     0,
}

const CMaxNumExps = 10

var cfgOpMap map[string]int = map[string]int{
	CfgOpRmprefix: 1,
	CfgOpRmsuffix: 1,
	CfgOpRmtail:   1,
	CfgOpAlphaNum: 1,
	CfgOpLc:       1,
	CfgOpReplace:  3,
	CfgOpRegexp:   2,
	CfgOpAnd:      CMaxNumExps,
	CfgOpOr:       CMaxNumExps,
}

type ConfigExpression struct {
	Source string              `mapstructure:"source"`
	Op     string              `mapstructure:"op"`
	Exps   []*ConfigExpression `mapstructure:"exps"`
}

type ConfigAttribute struct {
	Exp    *ConfigExpression `mapstructure:"exp"`
	Rename string            `mapstructure:"rename"`
}

type ConfigProfile struct {
	ServiceGroup *ConfigAttribute   `mapstructure:"service_group"`
	Host         *ConfigAttribute   `mapstructure:"host"`
	Logbasename  *ConfigAttribute   `mapstructure:"logbasename"`
	Severity     *ConfigAttribute   `mapstructure:"severity"`
	Labels       []*ConfigAttribute `mapstructure:"labels"`
	Message      *ConfigAttribute   `mapstructure:"message"`
	Format       string             `mapstructure:"format"`
}

func keysForMap(mymap map[string]int) []string {
	keys := make([]string, len(mymap))
	i := 0
	for k := range mymap {
		keys[i] = k
		i++
	}
	return keys
}

func validateCfgString(idx int, name, value string, cfgMap map[string]int) error {
	if value != "" {
		_, ok := cfgMap[value]
		if !ok {
			return fmt.Errorf("profile %d invalid value %s for %s, supported values %v", idx, value, name, keysForMap(cfgMap))
		}
	}
	return nil
}

func validateProfileExp(idx int, name string, exp *ConfigExpression) error {
	if exp == nil {
		return nil
	}
	if (exp.Source == "" && exp.Op == "") || (exp.Source != "" && exp.Op != "") {
		return fmt.Errorf("profile %d invalid, must specify exactly one of source or operator", idx)
	}
	arr := strings.SplitN(exp.Source, ":", 2)
	if len(arr) > 0 {
		err := validateCfgString(idx, "source", arr[0], cfgSourceMap)
		if err != nil {
			return err
		}
	}
	err := validateCfgString(idx, "op", exp.Op, cfgOpMap)
	if err != nil {
		return err
	}
	if exp.Op != "" {
		numExps, _ := cfgOpMap[exp.Op]
		if numExps != CMaxNumExps && len(exp.Exps) != numExps {
			return fmt.Errorf("profile %d invalid number of expressions %d for op %s expecting %d", idx, len(exp.Exps), exp.Op, numExps)
		}
		if numExps == CMaxNumExps && len(exp.Exps) < 2 {
			return fmt.Errorf("profile %d invalid number of expressions %d for op %s expecting 2 or more", idx, len(exp.Exps), exp.Op)
		}
		if exp.Op == CfgOpRegexp &&
			strings.HasPrefix(exp.Exps[1].Source, CfgSourceLit) {
			_, err = regexp.Compile(exp.Exps[1].Source[len(CfgSourceLit)+1:])
			if err != nil {
				return fmt.Errorf("profile %d invalid value %s for %s, regular expression invalid", idx, exp.Exps[1].Source, name)
			}
		}
	}
	for _, exp2 := range exp.Exps {
		err = validateProfileExp(idx, "exps", exp2)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateProfileElem(idx int, name string, attribute *ConfigAttribute) error {
	if attribute == nil {
		return nil
	}
	err := validateProfileExp(idx, name, attribute.Exp)
	if err != nil {
		return err
	}
	return nil
}

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	for idx, profile := range cfg.Profiles {
		if err := validateProfileElem(idx, "service_group", profile.ServiceGroup); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "host", profile.Host); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "logbasename", profile.Logbasename); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "severity", profile.Severity); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "message", profile.Message); err != nil {
			return err
		}
		err := validateCfgString(idx, "format", profile.Format, cfgFormatMap)
		if err != nil {
			return err
		}
		for _, label := range profile.Labels {
			if err := validateProfileElem(idx, "labels", label); err != nil {
				return err
			}
		}
	}
	if cfg.SendBatchMaxSize > 0 && cfg.SendBatchMaxSize < cfg.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}
	return nil
}
