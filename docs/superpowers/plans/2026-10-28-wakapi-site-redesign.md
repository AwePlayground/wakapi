# Wakapi Site Redesign Implementation Plan

**Goal:** Give every Wakapi page a consistent modern data workspace design without changing tracking or backend behavior.

**Architecture:** Add a small shared stylesheet after the existing Tailwind bundle, then update page templates only where structure or semantic state requires it. Keep all chart IDs, form names and action URLs.

**Tech Stack:** Go HTML templates, Tailwind CSS 3, local CSS, existing Iconify bundle, Docker/GitHub Actions.

## Global Constraints

- No new runtime dependency or external font request.
- Keep WakaTime and Wakapi heartbeat collection unchanged.
- Preserve existing API, chart IDs, form field names, and routes.
- Verify at desktop and mobile widths before deployment.

## Tasks

1. **Shared foundation:** Add brand tokens, typography, surface, controls, focus and navigation styling. Rebuild CSS. Verify the asset is served and templates parse.
2. **Dashboard and Projects:** Improve heading hierarchy, KPI cards, chart cards, project search and project items. Check data controls and project navigation.
3. **AI Overview:** Replace the old layout with a balanced hero, compact metrics, chart and breakdowns. Fix the missing menu icon with inline SVG. Check 7-day, 30-day and all-time pages.
4. **Settings and public pages:** Restyle tabs, forms, authentication, setup and leaderboard with the shared components. Verify form names/actions and responsive layout.
5. **Deploy:** Run tests and asset build, build image, stage-check the primary routes, back up production, update digest-pinned compose, verify health and pages, and fast-forward the fork's master branch.
