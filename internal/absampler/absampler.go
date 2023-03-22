// Copyright (c) 2023 ScienceLogic, Inc
package absampler

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/viki-org/dnscache"
)

var resolver *dnscache.Resolver
var transport *http.Transport

type samplerReq struct {
	SamplerVersion string   `json:"sampler_version"`
	Logbasename    string   `json:"logbasename"`
	Lines          []string `json:"lines"`
	SamplerPercent int      `json:"sampler_percent"`
}

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

// Create log lines
func createLogLines(info *AbSamplerInfo) ([]string, int) {

	rls := info.LD.ResourceLogs()
	initialLines := info.Config.SamplingInitial
	samplingTherafter := info.Config.SamplingThereafter
	numLogRecords := info.LD.LogRecordCount()

	if initialLines > numLogRecords {
		initialLines = numLogRecords
	}
	lines := make([]string, initialLines)
	if numLogRecords == 0 {
		return lines, 0
	}

	count := 0
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		ills := rl.ScopeLogs()
		for j := 0; j < ills.Len(); j++ {
			ils := ills.At(j)
			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				lr := logs.At(k)
				// Initial lines are sampled/preallocated.
				// Afterwards a sample of log lines is taken.
				if count < initialLines {
					lines[count] = lr.Body().AsString()
				} else if (count-initialLines)%samplingTherafter == 0 {
					lines = append(lines, lr.Body().AsString())
				}
				count++
			}
		}
	}
	if count == 0 {
		log.Printf("Got 0 lines, samplerPercent: 0")
		return lines, 0
	}
	samplerPercent := len(lines) * 100 / count
	log.Printf("Got %d lines of %d, samplerPercent: %d",
		len(lines), count, samplerPercent)
	if samplerPercent == 0 && len(lines) > 0 {
		samplerPercent = 1
	}
	return lines, samplerPercent
}

// Send logs
func sendSample(info *AbSamplerInfo) error {
	lines, samplerPercent := createLogLines(info)
	if len(lines) == 0 {
		// No data is available
		return ErrNoMoreLines
	}
	var body = &samplerReq{SamplerVersion: "0.0.5-otel",
		Logbasename:    info.Config.ZeLBN,
		Lines:          lines,
		SamplerPercent: samplerPercent}

	jsonValue, err := json.Marshal(body)
	if err != nil {
		log.Printf("Unable to marshal data for learner sample: %v", err)
		return err
	}

	url := info.Config.ZeURL + "/api/v2/learner"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Printf("Unable to get HTTP request for sample learner: %v", err)
		return err
	}
	client := &http.Client{Transport: transport}
	req.Header.Set("Authorization", "Token "+info.Config.ZeToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP error sending data: %v", err)
		return err
	}
	if resp == nil {
		log.Printf("No response from learner")
		return ErrResponseEmpty
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		log.Printf("Authorization error posting sample to Learner at %s: %s",
			url, resp.Status)
		return ErrBadAuthorization
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error getting HTTP response at %s: %v", url, err)
		return err
	}
	if resp.StatusCode > 299 {
		log.Printf("HTTP error %d posting sample to Learner at %s: %s",
			resp.StatusCode, url, respBody)
		return ErrHTTPPostFailed
	}
	log.Printf("Reponse '%s'", respBody)
	log.Printf("sent sampled lines: %d, bytes: %d, url: %s",
		len(lines), len(jsonValue), url)
	return nil
}

// Send a sample of work to Afterburner learner.
func SendWorkToAbSampler(samplerInfo *AbSamplerInfo) error {
	// Sending may need to be retried by higher layers, e.g due to
	// authentication errors.
	err := sendSample(samplerInfo)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	return nil
}
