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
	"encoding/json"
	"reflect"
	"testing"

	"github.com/wso2-open-operations/grc-tools/apps/grc-platform/backend/internal/risk/model"
	riskmysql "github.com/wso2-open-operations/grc-tools/apps/grc-platform/backend/internal/risk/repository/mysql"
)

// Parity for the risk repository's read paths.
//
// Create and Update are not covered: they write, and a test that creates risks
// would consume risk-code sequence numbers that never get released. They were
// verified by driving both implementations with the same request and comparing
// the responses, and through the UI.

func TestRiskListParity(t *testing.T) {
	skipUnlessParity(t)

	mysqlRepo := riskmysql.NewRiskRepository(parityDB(t))
	entityRepo := NewRiskRepository(parityClient(t))
	ctx := context.Background()

	// Each case exercises a different filter branch. The unfiltered case also
	// proves the two agree on ordering and paging, not just on contents.
	cases := []struct {
		name   string
		filter model.ListRisksFilter
	}{
		{"unfiltered", model.ListRisksFilter{Limit: 100}},
		{"by status", model.ListRisksFilter{Statuses: []string{"PENDING_RISK_OWNER_APPROVAL", "CLOSED"}, Limit: 100}},
		{"by level", model.ListRisksFilter{Levels: []string{"HIGH", "MEDIUM"}, Limit: 100}},
		{"by risk type", model.ListRisksFilter{RiskTypes: []string{"NEW"}, Limit: 100}},
		{"by search term", model.ListRisksFilter{Search: "a", Limit: 100}},
		{"overdue only", model.ListRisksFilter{DueOverdueOnly: true, Limit: 100}},
		{"submitted range", model.ListRisksFilter{SubmittedFrom: "2026-01-01", SubmittedTo: "2026-12-31", Limit: 100}},
		{"second page", model.ListRisksFilter{Limit: 5, Offset: 5}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want, err := mysqlRepo.List(ctx, c.filter)
			if err != nil {
				t.Fatalf("mysql List: %v", err)
			}
			got, err := entityRepo.List(ctx, c.filter)
			if err != nil {
				t.Fatalf("entity List: %v", err)
			}

			if want.Total != got.Total {
				t.Errorf("total: mysql %d, entity %d", want.Total, got.Total)
			}
			if len(want.Items) != len(got.Items) {
				t.Fatalf("items: mysql %d, entity %d", len(want.Items), len(got.Items))
			}
			for i := range want.Items {
				if !reflect.DeepEqual(want.Items[i], got.Items[i]) {
					t.Errorf("item %d differs:\n  mysql  %s\n  entity %s",
						i, asJSON(t, want.Items[i]), asJSON(t, got.Items[i]))
				}
			}
		})
	}
}

// TestRiskGetByIDParity compares the whole composite: resolved names, both
// scores, compliance references, the action plan with its steps, and the
// assessment log including the synthetic gross-score entry the backend
// prepends. This is the widest single mapping in the migration.
func TestRiskGetByIDParity(t *testing.T) {
	skipUnlessParity(t)

	db := parityDB(t)
	mysqlRepo := riskmysql.NewRiskRepository(db)
	entityRepo := NewRiskRepository(parityClient(t))
	ctx := context.Background()

	ids := riskIDsForParity(t, db)
	for _, id := range ids {
		want, err := mysqlRepo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("risk %d: mysql GetByID: %v", id, err)
		}
		got, err := entityRepo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("risk %d: entity GetByID: %v", id, err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Errorf("risk %d differs:\n  mysql  %s\n  entity %s",
				id, asJSON(t, want), asJSON(t, got))
		}
	}
	if !t.Failed() {
		t.Logf("%d risk details identical", len(ids))
	}
}

func TestRiskWorkflowStatusAndSequenceParity(t *testing.T) {
	skipUnlessParity(t)

	db := parityDB(t)
	mysqlRepo := riskmysql.NewRiskRepository(db)
	entityRepo := NewRiskRepository(parityClient(t))
	ctx := context.Background()

	for _, id := range riskIDsForParity(t, db) {
		want, err := mysqlRepo.GetWorkflowStatus(ctx, id)
		if err != nil {
			t.Fatalf("risk %d: mysql GetWorkflowStatus: %v", id, err)
		}
		got, err := entityRepo.GetWorkflowStatus(ctx, id)
		if err != nil {
			t.Fatalf("risk %d: entity GetWorkflowStatus: %v", id, err)
		}
		if want != got {
			t.Errorf("risk %d status: mysql %q, entity %q", id, want, got)
		}
	}

	// NextSequenceID previews without consuming, so it is safe to call here.
	rows, err := db.QueryContext(ctx, "SELECT id FROM risk_team WHERE status = 'ACTIVE' ORDER BY id")
	if err != nil {
		t.Fatalf("list registers: %v", err)
	}
	defer rows.Close()
	var registers []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan register id: %v", err)
		}
		registers = append(registers, id)
	}
	if len(registers) == 0 {
		t.Fatal("no active registers — the comparison would prove nothing")
	}
	for _, reg := range registers {
		want, err := mysqlRepo.NextSequenceID(ctx, reg)
		if err != nil {
			t.Fatalf("register %d: mysql NextSequenceID: %v", reg, err)
		}
		got, err := entityRepo.NextSequenceID(ctx, reg)
		if err != nil {
			t.Fatalf("register %d: entity NextSequenceID: %v", reg, err)
		}
		if want != got {
			t.Errorf("register %d next sequence: mysql %d, entity %d", reg, want, got)
		}
	}
}

// asJSON renders a value for failure messages. The models are deeply nested and
// full of pointers, so %+v prints addresses where the differing values should be.
func asJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return "<unmarshalable>"
	}
	return string(b)
}
