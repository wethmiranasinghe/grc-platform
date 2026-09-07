// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"testing"
	"time"
)

func TestEnvBool_WriteGateDefaultsToDryRun(t *testing.T) {
	// DRY_RUN defaults true — the tool must be told explicitly to write.
	cases := map[string]struct {
		set  bool
		val  string
		def  bool
		want bool
	}{
		"unset keeps default true":  {false, "", true, true},
		"unset keeps default false": {false, "", false, false},
		"true":                      {true, "true", true, true},
		"TRUE mixed case":           {true, "TRUE", true, true},
		"1":                         {true, "1", true, true},
		"yes":                       {true, "yes", true, true},
		"false":                     {true, "false", true, false},
		"anything else is false":    {true, "maybe", true, false},
		"empty string is default":   {true, "", true, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.set {
				t.Setenv("DRY_RUN_TEST_KEY", tc.val)
			}
			if got := envBool("DRY_RUN_TEST_KEY", tc.def); got != tc.want {
				t.Errorf("envBool(%q, %v) = %v, want %v", tc.val, tc.def, got, tc.want)
			}
		})
	}
}

func TestEnvDuration(t *testing.T) {
	if got := envDuration("MISSING_DURATION_KEY", 90*time.Second); got != 90*time.Second {
		t.Errorf("missing key should return the default, got %v", got)
	}
	t.Setenv("DURATION_KEY", "2m30s")
	if got := envDuration("DURATION_KEY", time.Second); got != 2*time.Minute+30*time.Second {
		t.Errorf("envDuration = %v", got)
	}
	t.Setenv("DURATION_KEY", "not-a-duration")
	if got := envDuration("DURATION_KEY", 5*time.Second); got != 5*time.Second {
		t.Errorf("an unparseable value should fall back to the default, got %v", got)
	}
}

func TestEnvOr(t *testing.T) {
	if got := envOr("UNSET_KEY_XYZ", "fallback"); got != "fallback" {
		t.Errorf("envOr unset = %q", got)
	}
	t.Setenv("SET_KEY_XYZ", "value")
	if got := envOr("SET_KEY_XYZ", "fallback"); got != "value" {
		t.Errorf("envOr set = %q", got)
	}
}
