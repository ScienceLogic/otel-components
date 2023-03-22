package otlpzapi

import (
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

var (
	ErrNoMoreLines      = errors.New("Log lines cannot be created")
	ErrBadAuthorization = errors.New("Bad Authorization")
	ErrNoMoreLbns       = errors.New("All LBNs are used")
	ErrStreamTokenEmpty = errors.New("Generated streamToken is empty")
	ErrResponseEmpty    = errors.New("Response is empty")
)

type ZapiLogConfig struct {
	ZeLBN   string
	ZeURL   string
	ZeToken string
}

type ZapiLogInfo struct {
	Config *ZapiLogConfig
	Logger *zap.Logger
	LD     *plog.Logs
}
