package sllogformatprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestMatchProfileSkipLogic(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	// Assert type assertion for *Config
	config, ok := cfg.(*Config)
	if !ok {
		t.Fatalf("Expected *Config but got %T", cfg)
	}

	// Update the config with the profile information
	config.Profiles = []ConfigProfile{
		{
			ServiceGroup: &ConfigAttribute{
				Rename: "service_group",
				Exp: &ConfigExpression{
					Source: "lit:default-group",
				},
			},
			Host: &ConfigAttribute{
				Rename: "host",
				Exp: &ConfigExpression{
					Source: "lit:test-host",
				},
			},
			Logbasename: &ConfigAttribute{
				Rename: "logbasename",
				Exp: &ConfigExpression{
					Source: "lit:example-log",
				},
			},
			Severity: &ConfigAttribute{
				Exp: &ConfigExpression{
					Source: "lit:INFO",
				},
			},
			Message: &ConfigAttribute{
				Exp: &ConfigExpression{
					Source: "body", // Message comes from the log body
				},
			},
		},
	}

	originalProfile := config.Profiles[0]

	logger := zap.NewNop()
	mockResourceLogs := plog.NewResourceLogs()
	mockScopeLogs := mockResourceLogs.ScopeLogs().AppendEmpty()

	// Table-driven test cases
	testCases := []struct {
		name          string
		ServiceGroup  string
		Host          string
		Logbasename   string
		Severity      string
		Message       string
		expectError   bool
		specificError error
	}{
		{
			name:          "All fields valid",
			ServiceGroup:  "default-group",
			Host:          "test-host",
			Logbasename:   "example-log",
			Severity:      "INFO",
			Message:       "valid log line",
			expectError:   false,
			specificError: nil,
		},
		{
			name:          "Empty service_group",
			ServiceGroup:  "",
			Host:          "test-host",
			Logbasename:   "example-log",
			Severity:      "INFO",
			Message:       "valid log line",
			expectError:   true,
			specificError: nil,
		},
		{
			name:          "Empty host",
			ServiceGroup:  "default-group",
			Host:          "",
			Logbasename:   "example-log",
			Severity:      "INFO",
			Message:       "valid log line",
			expectError:   true,
			specificError: nil,
		},
		{
			name:          "Empty logbasename",
			ServiceGroup:  "default-group",
			Host:          "test-host",
			Logbasename:   "",
			Severity:      "INFO",
			Message:       "valid log line",
			expectError:   true,
			specificError: nil,
		},
		{
			name:         "Empty severity",
			ServiceGroup: "default-group",
			Host:         "test-host",
			Logbasename:  "example-log",
			Severity:     "",
			Message:      "valid log line",
			// By setting the severity config to nil (below) we are asking the match
			// to use the log stream's severity number which does not generate an error.
			expectError:   false,
			specificError: nil,
		},
		{
			name:          "Empty message",
			ServiceGroup:  "default-group",
			Host:          "test-host",
			Logbasename:   "example-log",
			Severity:      "INFO",
			Message:       "",
			expectError:   true,
			specificError: errEmptyLine,
		},
		{
			name:          "Log line with unprintable characters",
			ServiceGroup:  "default-group",
			Host:          "test-host",
			Logbasename:   "example-log",
			Severity:      "INFO",
			Message:       "\x00\x01\x02\x03",
			expectError:   true,
			specificError: errEmptyLine,
		},
	}

	// Iterate through test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Modify the first profile based on the test case
			if tc.ServiceGroup == "" {
				config.Profiles[0].ServiceGroup = nil
			}
			if tc.Host == "" {
				config.Profiles[0].Host = nil
			}
			if tc.Logbasename == "" {
				config.Profiles[0].Logbasename = nil
			}
			if tc.Severity == "" {
				config.Profiles[0].Severity = nil
			}
			if tc.Message == "" {
				config.Profiles[0].Message = nil
			}

			// Create log record for the test case
			createLogRecord(mockScopeLogs, tc.Message)

			// Get the last log record (the one just created), and attempt to match a profile
			logRecord := mockScopeLogs.LogRecords().At(mockScopeLogs.LogRecords().Len() - 1)
			gen, req, err := config.MatchProfile(logger, mockResourceLogs, mockScopeLogs, logRecord)

			// Validate the error based on test case expectations
			if tc.expectError {
				if tc.specificError != nil {
					assert.ErrorAs(t, err, &tc.specificError, "expected specific error for log record")
				} else {
					assert.Error(t, err, "expected some error for log record")
				}
				assert.Empty(t, gen, "gen should be empty when log record is skipped")
				assert.Empty(t, req, "req should be empty when log record is skipped")
			} else {
				assert.NoError(t, err, "expected no error for valid log record")
				assert.NotEmpty(t, gen, "gen should not be empty for valid log record")
				assert.NotEmpty(t, req, "req should not be empty for valid log record")
			}

			// Restore the original profile
			config.Profiles[0] = originalProfile
		})
	}

}

// Creates a log record with specific attributes and log line
func createLogRecord(scopeLogs plog.ScopeLogs, logLine string) {
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr(logLine)
}
