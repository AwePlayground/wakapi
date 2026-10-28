# AI Overview Implementation Plan

> **For agentic workers:** Use the approved design in `docs/superpowers/specs/2026-10-28-ai-overview-design.md` and implement tasks sequentially with focused verification.

**Goal:** Deliver a private Wakapi AI activity dashboard with historical AI-field backfill and server-side estimated pricing.

**Architecture:** Preserve WakaTime AI heartbeat fields, aggregate from SQLite in the Go server, render the dashboard in Wakapi templates, and refresh public prices daily on the VPS.

**Tech Stack:** Go 1.27, GORM/SQLite, Go templates, Python 3 standard library, Docker Compose.

## Global Constraints

- Do not change local WakaTime dual-write configuration or add a local daemon.
- Do not query pricing APIs on each dashboard request.
- Label all costs as estimates and report unpriced tokens.
- Keep the page authenticated and do not expose source-code contents.

### Task 1: Preserve AI heartbeat fields

**Files:** `models/heartbeat.go`, `models/compat/wakatime/v1/heartbeat.go`, `services/imports/wakatime_utils.go`, focused tests.

- [ ] Write a failing importer round-trip test for token, prompt, session and line fields.
- [ ] Add cached input token persistence and map the export fields.
- [ ] Run focused tests and inspect schema migration.

### Task 2: Aggregate AI activity

**Files:** `services/ai_overview.go`, focused tests.

- [ ] Write failing tests for timezone boundaries, signed lines, tokens and grouped breakdowns.
- [ ] Implement bounded SQL aggregation for last 7, last 30 and all time.
- [ ] Verify with representative imported heartbeats.

### Task 3: Render the private dashboard

**Files:** `routes/ai.go`, `views/ai.tpl.html`, `views/menu-main.tpl.html`, `config/templates.go`, `main.go`.

- [ ] Write an authenticated route test.
- [ ] Add `/ai` with metric cards, daily chart, model and project sections, range selector, pricing coverage and empty states.
- [ ] Verify template rendering and responsive layout.

### Task 4: Refresh estimated pricing

**Files:** `deploy/refresh-ai-pricing.py`, `services/ai_pricing.go`, focused tests, VPS timer/config.

- [ ] Write tests for price parsing, aliasing, cached-input formula and missing prices.
- [ ] Fetch LiteLLM, models.dev and OpenRouter on the VPS daily into an atomic cache file.
- [ ] Expose cached estimate and provenance to the dashboard without request-time network calls.

### Task 5: Backfill, deploy and verify

**Files:** `deploy/backfill-ai-fields.py`, GitHub Actions workflow, VPS deployment.

- [ ] Prepare a dry-run backfill keyed by `origin_id`, compare counts and make a fresh backup.
- [ ] Build, test, push, and deploy a pinned image from the fork.
- [ ] Run backfill, verify sample day totals, cost coverage, existing project page and service health.
