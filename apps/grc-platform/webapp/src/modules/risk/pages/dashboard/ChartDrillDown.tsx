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

import { Box, Typography } from "@wso2/oxygen-ui";
import type { JSX, ReactNode } from "react";
import type { DrillDownFilter, OnDrillDown } from "./constants";

export interface DrillDownTarget {
  // Stable per segment; also the React key.
  key: string;
  // Read aloud and shown when the button takes focus, so it has to name the
  // segment on its own: "High, in Asgardeo", not "High".
  label: string;
  filter: DrillDownFilter;
}

interface ChartDrillDownProps {
  targets: DrillDownTarget[];
  onDrillDown: OnDrillDown;
  // What the buttons drill into, e.g. "risk level". Used for the group label.
  what: string;
  children: ReactNode;
}

// Keyboard and screen-reader access to a chart's click-to-filter.
//
// The charts' segments cannot carry this themselves. Recharts gives the
// <svg> surface one tab stop (arrow keys walk the tooltip along it) but leaves
// the individual bars and slices as bare <path> elements — no tabIndex, no
// role, no name — so there is nothing to focus and nothing for Enter to
// activate. Handing the series an onKeyDown does not help either: the Oxygen
// wrapper maps a fixed prop list onto recharts (dataKey, fill, label, onClick,
// the onMouse* family) and silently drops anything else, so such a handler
// would look right, typecheck, and never fire.
//
// So the accessible control is a real <button> per segment, rendered beside the
// chart rather than inside it. Each is off-screen until focused and then shown,
// which keeps the visual design untouched while giving keyboard users the same
// per-segment drill-down a mouse user gets by clicking — and giving screen
// reader users a list of the segments, which the <svg> alone never exposed.
export default function ChartDrillDown({
  targets,
  onDrillDown,
  what,
  children,
}: ChartDrillDownProps): JSX.Element {
  return (
    <Box
      sx={{
        position: "relative",
        // The segments are clickable but recharts renders them with the default
        // cursor, so nothing tells a mouse user they can be clicked.
        "& .recharts-bar-rectangle, & .recharts-pie-sector, & .recharts-sector": {
          cursor: "pointer",
        },
      }}
    >
      {children}
      {/* role="list"/"listitem" look redundant but are not: WebKit strips the
          implicit list role from a list styled list-style: none (and the
          inline items push the same way), and with it the aria-label — so
          VoiceOver, the only screen reader on iOS, would announce neither the
          group nor its size. Explicit roles keep both. */}
      <Box
        component="ul"
        role="list"
        aria-label={`Filter Risk Register by ${what}`}
        sx={{ listStyle: "none", m: 0, p: 0 }}
      >
        {targets.map((t) => (
          <Box component="li" role="listitem" key={t.key} sx={{ display: "inline" }}>
            <Box
              component="button"
              type="button"
              onClick={() => onDrillDown(t.filter)}
              sx={{
                // Off-screen but focusable — not display:none, which would drop
                // it out of the tab order and defeat the point.
                //
                // The sizes are px STRINGS deliberately. sx reads a bare `1` as
                // a theme size, i.e. width: 1 means 100%, so the hidden button
                // covered the whole chart and swallowed every click on a bar —
                // keyboard support that silently removed mouse support.
                position: "absolute",
                width: "1px",
                height: "1px",
                padding: 0,
                margin: "-1px",
                overflow: "hidden",
                clip: "rect(0, 0, 0, 0)",
                whiteSpace: "nowrap",
                border: 0,
                // Revealed on focus, but still absolutely positioned: letting it
                // back into the flow would reflow the card and make the chart
                // jump as a keyboard user tabs along the segments.
                "&:focus-visible": {
                  width: "auto",
                  height: "auto",
                  bottom: 0,
                  left: 0,
                  margin: 0,
                  px: 1,
                  py: 0.5,
                  clip: "auto",
                  overflow: "visible",
                  borderRadius: 1,
                  cursor: "pointer",
                  zIndex: 1,
                  bgcolor: "var(--oxygen-palette-background-default)",
                  outline: "2px solid currentColor",
                },
              }}
            >
              <Typography component="span" variant="caption">
                Show {t.label} in Risk Register
              </Typography>
            </Box>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
