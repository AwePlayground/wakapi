# Wakapi Site Redesign

## Scope

Redesign the whole user-facing Wakapi UI in the AwePlayground fork: Dashboard and project detail, Projects, AI Overview, Settings, Leaderboard, Setup, and public authentication pages. Preserve routes, API behavior, tracking, existing chart data, and the current green Wakapi brand. This is a visual and interaction update, not a backend rewrite.

## Direction

Use a modern data workspace style: deep neutral navy background, calm layered surfaces, clear borders, green for actions and AI activity, restrained secondary colors for metrics. Replace the current dense, uneven blocks with consistent 8px-based spacing, 12–16px radii, clearer page headings, compact navigation, and tabular numeric emphasis. Use system fonts and local SVGs; add no third-party runtime requests.

## Components

- Global tokens and reusable CSS for navigation, cards, controls, forms, tabs, charts, dropdowns, footer, hover and focus states.
- Page-specific layout rules for the Dashboard, Projects, Settings, AI Overview, Leaderboard, Setup, and public pages.
- Each primary page gets one clear heading and description; secondary text has consistent contrast and line length.
- Mobile layout keeps navigation usable, cards in one or two columns, and avoids horizontal scrolling.

## Data and behavior

Keep all existing statistics and data sources. The AI Overview retains its current price coverage and model assumptions. Project links, filters, date controls, settings forms, and authentication keep their existing behavior.

## Accessibility and performance

Keep semantic headings and visible labels. Focus indicators are always visible, selected navigation and tabs have text or `aria-current` signals, and controls have at least 44px touch targets where feasible. Respect reduced-motion settings. Styles are local CSS and existing assets, with no extra data fetch or background work.

## Verification

Build Tailwind and icons, run template and Go tests, inspect desktop/mobile markup and computed CSS in a local or staging render, then deploy a digest-pinned image after a production backup. Verify all main routes and key forms on the VPS; retain the prior compose file and image digest for rollback.
