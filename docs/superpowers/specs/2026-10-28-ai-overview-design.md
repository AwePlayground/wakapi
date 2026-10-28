# AI Overview Design

## Purpose

Add a private, self-hosted AI activity dashboard to the AwePlayground Wakapi fork. It reads Wakapi heartbeats and does not add a local background agent or depend on WakaTime during normal use.

## Data and accuracy

The dashboard shows AI and human line changes, input, cached input and output tokens, prompt events, AI sessions, daily activity, and model and project breakdowns for the current user. It supports the last 7 days, last 30 days and all time in the user's Wakapi timezone. Historical WakaTime heartbeats are backfilled from the user's existing export using origin IDs; unmatched rows are reported and skipped. The WakaTime importer and heartbeat API preserve these fields going forward.

Costs are **estimates**, never invoices. The VPS refreshes public model price catalogs once daily, following the user's Tokscale sources: LiteLLM, models.dev and OpenRouter. Cost is calculated from uncached input, cached input, and output tokens with explicit model aliases. Generic model family names require an assumption; the page displays source, refresh time, mapped/unmapped token coverage, and distinguishes unpriced activity. A missing price never becomes a zero-dollar claim.

WakaTime's global-user comparison, human review sessions, and follow-up sessions are excluded because their exact definitions or population data cannot be reconstructed from self-hosted heartbeats. The dashboard labels these omissions instead of inventing equivalent numbers.

## Architecture

- Extend the heartbeat model and WakaTime compatibility/import model with cached input tokens and preserve all existing AI fields in imports.
- Add a query service that performs indexed SQL aggregates over heartbeats for the selected interval and caches results briefly. Never load all raw heartbeats into a web request.
- Add a web-authenticated `/ai` page integrated into Wakapi's existing menu and template system. Render with server-side HTML and lightweight CSS bars; no new browser framework.
- Add a server-side price refresh script and daily timer. Catalog fetches occur on the VPS only, with cached prices stored outside the application database. The app tolerates stale or absent pricing data.
- Add a one-time historical backfill tool using the user's 2026-10-28 WakaTime export and the imported `origin_id` values, with backup and verification before production updates.

## Release and verification

Implement on a feature branch in AwePlayground/wakapi. Run focused Go tests and build the container in GitHub Actions. Back up the VPS database, deploy the pinned image, run the backfill, and compare selected AI metrics against WakaTime's export. Verify regular WakaTime and Wakapi dual write, project list and service health remain intact.
