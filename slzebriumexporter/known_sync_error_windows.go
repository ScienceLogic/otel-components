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

//go:build windows
// +build windows

package slzebriumexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/slzebriumexporter"

import "golang.org/x/sys/windows"

// knownSyncError returns true if the given error is one of the known
// non-actionable errors returned by Sync on Windows:
//
// - sync /dev/stderr: The handle is invalid.
func knownSyncError(err error) bool {
	return err == windows.ERROR_INVALID_HANDLE
}
