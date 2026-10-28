# AI Overview

The private `/ai` page summarizes AI fields from Wakapi heartbeats for the logged-in user. It needs no additional editor plugin. The page supports 7 days, 30 days and all time, using the user's Wakapi timezone.

## Metrics

Input tokens are uncached input plus cached input. Signed line changes are shown as additions and deletions. Prompt events count heartbeats with a positive `ai_prompt_length`; AI sessions count distinct nonempty `ai_session` IDs. Missing fields are not inferred from ordinary coding activity. The page excludes WakaTime's global-user comparison, human-review rate and follow-up rate because the self-hosted heartbeats do not contain their exact definitions.

## Cost estimates

`deploy/refresh_ai_pricing.py` fetches public price catalogs from LiteLLM, models.dev and OpenRouter, in that priority order, and writes an atomic normalized cache. The VPS systemd timer refreshes it daily. The application reads the cached file at `WAKAPI_AI_PRICING_FILE` (default `/data/ai-pricing.json`) and does no request-time price fetch.

`deploy/ai-model-aliases.json` maps generic heartbeat labels to specific list-price models. The included Opus, Sonnet and Gpt aliases are **assumptions** and should be reviewed when the actual model version changes. Cached input is priced separately; missing model or rate information is counted as unpriced. The page shows pricing coverage and labels every amount as an estimate at current list prices, not a bill or a historical price reconstruction.

For a VPS bind mount, mount `/opt/wakapi/pricing` read-only into the container at `/pricing`, set `WAKAPI_AI_PRICING_FILE=/pricing/ai-pricing.json`, install the refresh script and alias file at `/opt/wakapi`, and enable `deploy/wakapi-ai-pricing.timer` and `.service` under `/etc/systemd/system`. A failed refresh leaves the previous cache intact.

## Historic data

WakaTime's raw heartbeat export has AI fields that older Wakapi imports did not preserve. `deploy/backfill_ai_fields.py extract` creates a compact SQLite staging file from an export before a chosen cutoff timestamp. `apply` first reports how many imported Wakapi rows match by `origin_id`; `apply --apply --backup PATH` creates and checks a database backup, then updates the matched AI fields. Deploy the schema migration before applying the backfill. The script does not touch live heartbeats newer than the cutoff because the staging file excludes them.
