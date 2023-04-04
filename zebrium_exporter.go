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

package slzebriumexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/slzebriumexporter"

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/pdata/plog"
)

func (s *loggingExporter) getStreamTokenRequest(rl plog.ResourceLogs) (string, error) {
	val, ok := rl.Resource().Attributes().Get("sl_metadata")
	request := val.AsString()
	if !ok || request == "" {
		return "", errors.New("Missing sl_metadata resource attribute, configure log format processor")
	}
	return request, nil
}

func (s *loggingExporter) getStreamToken(request string) (string, error) {
	url := s.cfg.Endpoint + "/api/v2/token"
	req, err := http.NewRequest("POST", url, strings.NewReader(request))
	if err != nil {
		s.log.Error("Unable to get HTTP request for stream token", zap.String("err", err.Error()))
		return "", err
	}
	req.Header.Set("Authorization", "Token "+s.cfg.ZeToken)
	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Error("HTTP error getting stream token", zap.String("err", err.Error()))
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.log.Error("Error getting HTTP response", zap.String("err", err.Error()))
		return "", err
	}
	type RespToken struct {
		Token string `json:"token"`
	}
	var respToken RespToken
	err = json.Unmarshal(body, &respToken)
	if err != nil {
		s.log.Error("Unable to unmarshal token response", zap.String("err", err.Error()),
			zap.String("body", string(body)))
		return "", err
	}
	token := respToken.Token
	if token == "" {
		s.log.Error("Got empty stream token", zap.String("body", string(body)))
		return "", errors.New("Got empty stream token")
	}
	return token, nil
}

func (s *loggingExporter) sendLogs(request, format string, buffer []byte) error {
	alreadyRetried := false
	token, _ := s.streamTokenMap[request]
retry:
	if token == "" {
		var err error
		token, err = s.getStreamToken(request)
		if err != nil {
			return err
		}
	}
	url := s.cfg.Endpoint
	switch format {
	case CfgFormatEvent:
		url += "/api/v2/tmpost"
	default:
		url += "/api/v2/post"
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(buffer))
	if err != nil {
		s.log.Info("Unable to get HTTP request for stream token", zap.String("err", err.Error()))
		return err
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Close = true
	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Info("HTTP error sending data", zap.String("err", err.Error()))
		return err
	}
	if resp == nil {
		err = errors.New("Response is empty")
		s.log.Info("HTTP error sending data", zap.String("err", err.Error()))
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		s.log.Error("Authorization error posting to Zapi", zap.String("status", resp.Status))
		if !alreadyRetried {
			s.log.Info("Refresh token ...")
			token = ""
			alreadyRetried = true
			goto retry
		}
		return errors.New("Bad Authorization")
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		s.log.Info("Error getting HTTP response", zap.String("err", err.Error()))
		return err
	}
	return nil
}

func (s *loggingExporter) marshalLogs(rl plog.ResourceLogs) ([]byte, error) {
	out := []byte{}
	ills := rl.ScopeLogs()
	for j := 0; j < ills.Len(); j++ {
		ils := ills.At(j)
		logs := ils.LogRecords()
		for k := 0; k < logs.Len(); k++ {
			lr := logs.At(k)
			val, ok := lr.Attributes().Get("sl_msg")
			msg := val.AsString()
			if !ok || msg == "" {
				return []byte{}, errors.New("Missing sl_msg log record attribute, configure log format processor")
			}
			out = append(out, (msg + "\n")...)
		}
	}
	return out, nil
}
