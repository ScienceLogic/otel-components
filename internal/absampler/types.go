// Copyright (c) 2023 ScienceLogic, Inc
package absampler

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var (
	ErrNoMoreLines      = errors.New("Log lines cannot be created")
	ErrBadAuthorization = errors.New("Bad Authorization")
	ErrResponseEmpty    = errors.New("Response is empty")
	ErrHTTPPostFailed   = errors.New("HTTP error in POST")
)

type AbSamplerConfig struct {
	ZeLBN              string
	ZeURL              string
	ZeToken            string
	SamplingInitial    int // Number of initial log records to sample, >=0
	SamplingThereafter int // Sampling percentage after initial sample
}

type AbSamplerInfo struct {
	Config *AbSamplerConfig
	Logger *zap.Logger
	LD     *plog.Logs
}
