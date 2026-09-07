# risk-register-import

One-time migration that loads the historical risk register into the platform
database by driving the **compliance-entity HTTP API**. Deployed as a **Choreo
Manual Task** (Go).

Full design: `RISK_REGISTER_CSV_MIGRATION_PLAN.md` (a planning doc kept outside
this repo). Section references (§n) in the code comments point there.

## What it does

1. **Preflight** (§7) — entity health, SCIM reachable, the three risk roles
   `ACTIVE`, reference data populated, CSV headers present. Any failure aborts
   before a single write.
2. **Parse + map** the CSV (§4, §6) — header-name matched, legend/blank rows
   skipped, values transformed, bad rows → `REJECT` findings.
3. **Resolve people** (§6) — work email → uuid via the SCIM org snapshot →
   `user.id` via `GET /users/by-uuid` (else `POST /users`, never mutating an
   existing row).
4. **Dry run** (default) stops here and prints the report.
5. **Reconstruct resume state** from the entity (§8) — no ledger file.
6. **Write** each row in `Migration ID` order (§5):
   - `POST /risks` (born `PENDING_RISK_OWNER_APPROVAL`, `created_by` = marker)
   - walk `workflow_status` to the row's bucket via `PATCH` (no-op when
     `from == to`)
   - `IN_REMEDIATION` + overdue → `POST /risks/{id}/escalations` to suppress the
     nightly escalation job
   - `IN_REMEDIATION` → ensure the §6 role grants
   - `CLOSED` → also `PATCH` the action plan to `COMPLETED`

## Buckets

Every row's added `Workflow Status` column is exactly `IN_REMEDIATION` or
`CLOSED`. Anything else rejects the row. There is no brand-new / `DRAFT` /
`CANCELLED` bucket.

## Run it

Local (against reachable staging URLs):

```
COMPLIANCE_ENTITY_BASE_URL=... \
SCIM_BASE_URL=https://api.asgardeo.io SCIM_INTERNAL_ORG=wso2 \
SCIM_INTERNAL_CLIENT_ID=... SCIM_INTERNAL_CLIENT_SECRET=... SCIM_INTERNAL_SCOPES=... \
go run . -input ./risks.csv -migration-date 2026-09-15          # dry run

# ... fix everything the report REJECTs, re-export, repeat ...

go run . -input ./risks.csv -migration-date 2026-09-15 -dry-run=false
```

`SCIM_TOKEN_URL` is optional — derived as
`{SCIM_BASE_URL}/t/{SCIM_INTERNAL_ORG}/oauth2/token`. `go run . -h` prints the
full flag surface; every `.choreo/component.yaml` config key maps to a `Config`
field in `main.go`.

In Choreo: set the config keys in `.choreo/component.yaml`, upload the CSV as
the `INPUT_PATH` file mount, leave `DRY_RUN=true`, trigger, read the logs; then
flip `DRY_RUN=false` and trigger again.

Exit codes: `0` clean · `2` completed with findings · `1` structural abort.

## Fully local end-to-end (no Asgardeo credentials)

`-scim-snapshot <file>` replaces the live Asgardeo call with an `email,uuid`
CSV, so the whole pipeline runs against a local entity + local MySQL with no
directory access. **Local testing only — never set `SCIM_SNAPSHOT_FILE` in
Choreo.** When it is set, the `SCIM_INTERNAL_*` config is not required.

```bash
# 1. local MySQL
mysql -e "CREATE DATABASE grc_platform"
mysql grc_platform < ../../apps/grc-platform/backend/Resources/shared.sql
mysql grc_platform < ../../apps/grc-platform/backend/Resources/risk_schema.sql
mysql grc_platform < ../../apps/grc-platform/backend/Resources/shared_seed_data.sql   # risk roles
mysql grc_platform < <path-to>/risk_module_data_schema.sql                            # teams / categories / refs / scores

# 2. compliance-entity (separate shell, leave running)
cd ../../entity/compliance-entity
printf 'DB_DSN=root:@tcp(127.0.0.1:3306)/grc_platform?parseTime=true\nSERVER_PORT=8080\n' > .env
go run ./cmd/api

# 3. this tool — dry run, then real run
cd ../../operations/risk-register-import
IN='<planning-docs>/Risk_Test_Full.xlsx - New Risk Form Structure.csv'
SNAP='<planning-docs>/scim-snapshot.csv'

COMPLIANCE_ENTITY_BASE_URL=http://localhost:8080 \
go run . -input "$IN" -scim-snapshot "$SNAP" -migration-date 2026-09-15

COMPLIANCE_ENTITY_BASE_URL=http://localhost:8080 \
go run . -input "$IN" -scim-snapshot "$SNAP" -migration-date 2026-09-15 -dry-run=false
```

Inspect: `SELECT risk_code, workflow_status, treatment_strategy FROM risk WHERE created_by='risk-sheet-migration'`.
Reset with `rollback.sql`; a second real run must be all-`Skipped`.

The shipped test pair lives in the planning-docs area (out of git):
`Risk_Test_Full.xlsx - New Risk Form Structure.csv` (5 valid rows: 3
`IN_REMEDIATION`, 2 `CLOSED`, one `ACCEPT`/high row that also gets the
management grant) and `scim-snapshot.csv` (its five emails → placeholder uuids).

## Report

The run prints two blocks to stdout (captured by Choreo):

- **`errors.csv`** — one row per finding: `migration_id, csv_row, risk_title,
  severity (REJECT|WARN), failure, detail`.
- **`report.txt`** — findings-by-code counts, the distinct unresolved people,
  the Migration IDs that got a suppressing escalation, the grant count, and
  (real run) per-bucket migrated counts.

## Rollback

`rollback.sql` — by-marker `DELETE` in FK order, run by hand against the target
DB. The tool never deletes.

## Status

Implemented (T1–T9, T11): HTTP + SCIM clients, preflight + reference data, CSV
value mapping, identity resolution, resume-state reconstruction, the write
pipeline, the report, and a table-driven test suite including an end-to-end
pass over `testdata/risks.csv` (built from the real "Risk Form Structure" tab)
against a stateful in-memory fake of the entity — including a second run that
must be a clean no-op. `go test ./...` is green (~80% statement coverage; the
gap is `main`/`run`/`loadConfig` CLI bootstrap).

Not yet done: `.choreo/component.yaml` confirmation as a real Manual Task in the
Choreo console (T10) and the staging/production dry-run → real-run passes (T12).
