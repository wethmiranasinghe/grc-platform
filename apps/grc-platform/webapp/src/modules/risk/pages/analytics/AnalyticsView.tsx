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

import { Box, Stack } from "@wso2/oxygen-ui";
import type { JSX } from "react";
import type { AnalyticsSummary } from "../../api/riskApi";
import ChartCard from "../dashboard/ChartCard";
import KpiTiles from "./KpiTiles";
import TrendChart from "./TrendChart";
import LevelDistributionChart from "./LevelDistributionChart";
import RegisterShareDonut from "./RegisterShareDonut";
import ComplianceDonut from "./ComplianceDonut";
import TreatmentRadial from "./TreatmentRadial";
import WorkflowFunnelChart from "./WorkflowFunnelChart";
import AgingRisksTable from "./AgingRisksTable";

interface AnalyticsViewProps {
  analytics: AnalyticsSummary;
  isAllRegisters: boolean;
}

// Pure layout of the Risk Analytics page; RiskAnalytics supplies fetched data
// and the register filter. Deliberately avoids re-showing what the Dashboard
// already covers as a point-in-time snapshot — every chart here either adds a
// time dimension or is an org-wide aggregate the Dashboard doesn't compute.
export default function AnalyticsView({ analytics, isAllRegisters }: AnalyticsViewProps): JSX.Element {
  return (
    <Stack spacing={3}>
      <KpiTiles loading={false} kpis={analytics.kpis} />

      <ChartCard title="Risk Trend Over Time">
        <TrendChart data={analytics.trend} />
      </ChartCard>

      <ChartCard title="Risk Level Distribution Over Time">
        <LevelDistributionChart data={analytics.level_distribution} />
      </ChartCard>

      <Box
        sx={{
          display: "grid",
          gridTemplateColumns: {
            xs: "1fr",
            md: isAllRegisters ? "repeat(3, 1fr)" : "repeat(2, 1fr)",
          },
          gap: 3,
        }}
      >
        {isAllRegisters && (
          <ChartCard title="Risks by Register">
            <RegisterShareDonut data={analytics.register_shares} />
          </ChartCard>
        )}
        <ChartCard title="Compliance Reference Distribution">
          <ComplianceDonut data={analytics.compliance_distribution} />
        </ChartCard>
        <ChartCard title="Risk Treatment Strategies">
          <TreatmentRadial data={analytics.treatment_mix} />
        </ChartCard>
      </Box>

      <ChartCard title="Workflow Status Funnel">
        <WorkflowFunnelChart data={analytics.workflow_funnel} />
      </ChartCard>

      <ChartCard title="Aging Open Risks">
        <AgingRisksTable data={analytics.aging_risks} />
      </ChartCard>
    </Stack>
  );
}
