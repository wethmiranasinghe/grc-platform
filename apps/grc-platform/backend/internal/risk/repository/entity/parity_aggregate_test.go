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

package entity

import (
	"context"
	"reflect"
	"testing"

	riskmysql "github.com/wso2-open-operations/grc-tools/apps/grc-platform/backend/internal/risk/repository/mysql"
	riskservice "github.com/wso2-open-operations/grc-tools/apps/grc-platform/backend/internal/risk/service"
)

// Parity for the two aggregate pages.
//
// These compare across an architectural boundary rather than between two
// implementations of one interface. On MySQL the repository returns fact rows
// and the *service* pivots them into charts; on the entity both happen
// server-side and the repository returns the finished payload. So the
// comparison is MySQL service output against entity repository output — which
// is exactly the thing that must not drift, since the two now compute the same
// charts from different code.
//
// Both sides read their own clock. `age_days` comes from CURDATE() and the
// analytics window from time.Now(), so a run that straddles midnight can
// legitimately differ by a day. If one of these fails on nothing but dates,
// re-run it before investigating.

func TestDashboardParity(t *testing.T) {
	skipUnlessParity(t)

	mysqlSvc := riskservice.NewDashboardService(riskmysql.NewDashboardRepository(parityDB(t)))
	entityRepo := NewDashboardRepository(parityClient(t))
	ctx := context.Background()

	for _, scope := range parityRegisterScopes(t) {
		t.Run(scope.name, func(t *testing.T) {
			want, err := mysqlSvc.Summary(ctx, scope.registerID)
			if err != nil {
				t.Fatalf("mysql dashboard: %v", err)
			}
			got, err := entityRepo.Summary(ctx, scope.registerID)
			if err != nil {
				t.Fatalf("entity dashboard: %v", err)
			}
			if !reflect.DeepEqual(want, got) {
				t.Errorf("dashboard differs:\n  mysql  %s\n  entity %s",
					asJSON(t, want), asJSON(t, got))
			}
		})
	}
}

func TestAnalyticsParity(t *testing.T) {
	skipUnlessParity(t)

	mysqlSvc := riskservice.NewAnalyticsService(riskmysql.NewAnalyticsRepository(parityDB(t)))
	entityRepo := NewAnalyticsRepository(parityClient(t))
	ctx := context.Background()

	for _, scope := range parityRegisterScopes(t) {
		t.Run(scope.name, func(t *testing.T) {
			want, err := mysqlSvc.Summary(ctx, scope.registerID)
			if err != nil {
				t.Fatalf("mysql analytics: %v", err)
			}
			got, err := entityRepo.Summary(ctx, scope.registerID)
			if err != nil {
				t.Fatalf("entity analytics: %v", err)
			}
			if !reflect.DeepEqual(want, got) {
				t.Errorf("analytics differs:\n  mysql  %s\n  entity %s",
					asJSON(t, want), asJSON(t, got))
			}
		})
	}
}

type parityScope struct {
	name       string
	registerID *int
}

// parityRegisterScopes returns the scopes both aggregate pages are compared
// under: unfiltered, plus one real register. The filtered case matters on its
// own — analytics omits the per-register series entirely when a filter is
// applied, so only that path proves nil is preserved rather than turned into an
// empty slice.
func parityRegisterScopes(t *testing.T) []parityScope {
	t.Helper()
	db := parityDB(t)

	var registerID int
	err := db.QueryRowContext(context.Background(),
		"SELECT source_register_id FROM risk GROUP BY source_register_id ORDER BY COUNT(*) DESC LIMIT 1").
		Scan(&registerID)
	if err != nil {
		t.Fatalf("pick a register with risks: %v", err)
	}
	return []parityScope{
		{"all registers", nil},
		{"single register", &registerID},
	}
}
