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
	CfgOptionRmprefix  string = "rmprefix"
	CfgOptionRmsuffix  string = "rmsuffix"
	CfgOptionRmtail    string = "rmtail"
	CfgOptionAlphaNum  string = "alphanum"
	CfgOptionLc        string = "lc"
)

var cfgIdNames map[string]struct{} = map[string]struct{}{
	"service_group": {},
	"host":          {},
	"logbasename":   {},
}

var cfgSourceMap map[string]struct{} = map[string]struct{}{
	CfgSourceRattr: {},
	CfgSourceAttr:  {},
	CfgSourceBody:  {},
	CfgSourceLit:   {},
}

var cfgFormatMap map[string]struct{} = map[string]struct{}{
	CfgFormatMessage:   {},
	CfgFormatContainer: {},
	CfgFormatEvent:     {},
}

var cfgOptionMap map[string]struct{} = map[string]struct{}{
	CfgOptionRmprefix: {},
	CfgOptionRmsuffix: {},
	CfgOptionRmtail:   {},
	CfgOptionAlphaNum: {},
	CfgOptionLc:       {},
}

type ConfigProfile struct {
	ServiceGroup string   `mapstructure:"service_group"`
	Host         string   `mapstructure:"host"`
	Logbasename  string   `mapstructure:"logbasename"`
	Severity     string   `mapstructure:"severity"`
	Labels       []string `mapstructure:"labels"`
	Message      string   `mapstructure:"message"`
	Format       string   `mapstructure:"format"`
}

func keysForMap(mymap map[string]struct{}) []string {
	keys := make([]string, len(mymap))
	i := 0
	for k := range mymap {
		keys[i] = k
		i++
	}
	return keys
}

func validateProfileElem(idx int, name, str string, cfgMap map[string]struct{}) error {
	arr := strings.Split(str, ":")
	if len(arr) < 1 || len(arr[0]) < 1 {
		return fmt.Errorf("profile %d missing %s", idx, name)
	}
	elem := arr[0]
	if strings.HasPrefix(elem, "replace(") {
		idx2 := strings.Index(elem, ",")
		if idx2 < 0 {
			return fmt.Errorf("profile %d invalid value %s for %s, replace requires arguments", idx, arr[0], name)
		}
		elem = elem[len("replace("):idx2]
		if len(elem) < 1 {
			return fmt.Errorf("profile %d invalid value %s for %s, replace requires arguments", idx, arr[0], name)
		}
	}
	arr3 := strings.SplitN(elem, ".", 2)
	if len(arr3) < 1 || len(arr3[0]) < 1 {
		return fmt.Errorf("profile %d missing %s", idx, name)
	}
	_, ok := cfgMap[arr3[0]]
	if !ok {
		return fmt.Errorf("profile %d invalid value %s for %s, supported values %v", idx, arr3[0], name, keysForMap(cfgMap))
	}
	_, ok = cfgIdNames[name]
	if ok && len(arr) < 2 {
		return fmt.Errorf("profile %d - %s: %s requires a replacement key", idx, name, str)
	}
	if len(arr) > 2 {
		for _, option := range arr[2:] {
			arr2 := strings.SplitN(option, "=", 2)
			if len(arr2) < 1 || len(arr2[0]) < 1 {
				return fmt.Errorf("profile %d missing option for %s", idx, name)
			}
			_, ok := cfgOptionMap[arr2[0]]
			if !ok {
				return fmt.Errorf("profile %d invalid option %s for %s, supported values %v", idx, arr2[0], name, keysForMap(cfgOptionMap))
			}
		}
	}
	return nil
}

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	for idx, profile := range cfg.Profiles {
		if err := validateProfileElem(idx, "service_group", profile.ServiceGroup, cfgSourceMap); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "host", profile.Host, cfgSourceMap); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "logbasename", profile.Logbasename, cfgSourceMap); err != nil {
			return err
		}
		if profile.Severity != "" {
			if err := validateProfileElem(idx, "severity", profile.Severity, cfgSourceMap); err != nil {
				return err
			}
		}
		if err := validateProfileElem(idx, "message", profile.Message, cfgSourceMap); err != nil {
			return err
		}
		if err := validateProfileElem(idx, "format", profile.Format, cfgFormatMap); err != nil {
			return err
		}
		for _, label := range profile.Labels {
			if err := validateProfileElem(idx, "labels", label, cfgSourceMap); err != nil {
				return err
			}
		}
	}
	if cfg.SendBatchMaxSize > 0 && cfg.SendBatchMaxSize < cfg.SendBatchSize {
		return errors.New("send_batch_max_size must be greater or equal to send_batch_size")
	}
	return nil
}
