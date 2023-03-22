package otlpzapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/viki-org/dnscache"
	"go.opentelemetry.io/collector/pdata/plog"
)

var resolver *dnscache.Resolver
var transport *http.Transport

func init() {
	resolver = dnscache.New(time.Minute * 5)
	transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Dial: func(network string, address string) (net.Conn, error) {
			separator := strings.LastIndex(address, ":")
			ip, _ := resolver.FetchOneString(address[:separator])
			return net.Dial("tcp", ip+address[separator:])
		},
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     30 * time.Second,
		WriteBufferSize:     32768,
	}
}

type MetaData struct {
	Logbasename string            `json:"logbasename"`
	Ids         map[string]string `json:"ids"`
	Cfgs        map[string]string `json:"cfgs"`
	Tags        map[string]string `json:"tags"`
}

// Gets a stream token, either from cache or by lookup from ZAPI
func getStreamToken(zapiInfo *ZapiLogInfo) (string, error) {
	cfg := zapiInfo.Config
	client := &http.Client{Transport: transport}
	ids := make(map[string]string)
	cfgs := make(map[string]string)
	tags := make(map[string]string)
	values := MetaData{Logbasename: cfg.ZeLBN, Ids: ids, Cfgs: cfgs, Tags: tags}

	jsonValue, err := json.Marshal(values)
	if err != nil {
		log.Printf("Unable to marshal data for stream token: %v", err)
		return "", err
	}
	log.Printf("metadata: %s", string(jsonValue))
	url := cfg.ZeURL + "/api/v2/token"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Printf("Unable to get HTTP request for stream token: %v", err)
		return "", err
	}
	req.Header.Set("Authorization", "Token "+cfg.ZeToken)
	req.Header.Set("Content-Type", "application/json")
	req.Close = true
	log.Printf("url: '%s', token: '%s'", url, cfg.ZeToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP error getting stream token: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error getting HTTP response: %v", err)
		return "", err
	}

	type RespToken struct {
		Token string `json:"token"`
	}
	var respToken RespToken
	err = json.Unmarshal(body, &respToken)
	if err != nil {
		log.Printf("Unable to unmarshal token: %v, reponse: %s", err, string(body))
		return "", err
	}
	token := respToken.Token
	if token == "" {
		log.Printf("Got empty stream token: '%s'", string(body))
		return "", errors.New("Got empty stream token")
	}
	return token, nil
}

// Create log lines
func createLogLines(lp *plog.Logs) (string, int32) {

	var sb strings.Builder
	var lines int32 = 0
	rls := lp.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		ills := rl.ScopeLogs()
		for j := 0; j < ills.Len(); j++ {
			ils := ills.At(j)
			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)
				sb.WriteString(lr.Body().AsString())
				sb.WriteString("\n")
				lines++
			}
		}
	}
	log.Printf("Got %d lines: '%s'", lines, sb.String())
	return sb.String(), lines
}

// Send logs
func sendLogs(info *ZapiLogInfo, streamToken string) error {
	if len(streamToken) == 0 {
		return ErrStreamTokenEmpty
	}

	body, lines := createLogLines(info.LD)
	if len(body) == 0 {
		// No data is available
		return ErrNoMoreLines
	}
	url := info.Config.ZeURL + "/api/v2/post"
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		log.Printf("Unable to get HTTP request for stream token: %v", err)
		return err
	}
	client := &http.Client{Transport: transport}
	req.Header.Set("Authorization", "Token "+streamToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP error sending data: %v", err)
		return err
	}
	if resp == nil {
		return ErrResponseEmpty
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		log.Printf("Authorization error posting to Zapi: %s",
			resp.Status)
		return ErrBadAuthorization
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error getting HTTP response: %v", err)
		return err
	}
	log.Printf("sent lines: %d, bytes: %d", lines, len(body))
	return nil
}

// Send a single piece of work to Zapi.
//func SendWorkToZapi(ld plog.Logs, lbn, zeUrl, zeToken string) error {
func SendWorkToZapi(zapiInfo *ZapiLogInfo) error {
	streamToken, err := getStreamToken(zapiInfo)
	if err != nil {
		log.Printf("%d: Unable to get stream token: %v", err)
		return err
	}

	// Sending may need to be retried by higher layers, e.g due to
	// authentication errors.
	err = sendLogs(zapiInfo, streamToken)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	return nil
}
