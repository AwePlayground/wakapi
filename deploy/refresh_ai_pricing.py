#!/usr/bin/env python3
"""Refresh Wakapi's server-side AI price cache from public catalogs."""

import argparse
import datetime
import json
import math
import os
import pathlib
import tempfile
import time
import urllib.request

SOURCES = {
    "litellm": "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
    "models.dev": "https://models.dev/api.json",
    "openrouter": "https://openrouter.ai/api/v1/models",
}


def number(value):
    try:
        value = float(value)
    except (TypeError, ValueError):
        return None
    return value if math.isfinite(value) and value >= 0 else None


def price(input_rate, cached_rate, output_rate, source):
    if input_rate is None and output_rate is None:
        return None
    return {"input": input_rate, "cached_input": cached_rate, "output": output_rate, "source": source}


def normalize_litellm(data):
    result = {}
    for model, entry in data.items():
        if not isinstance(entry, dict):
            continue
        value = price(number(entry.get("input_cost_per_token")),
                      number(entry.get("cache_read_input_token_cost")),
                      number(entry.get("output_cost_per_token")), "litellm")
        if value:
            result[model] = value
    return result


def normalize_models_dev(data):
    result = {}
    for provider, info in data.items():
        if not isinstance(info, dict):
            continue
        for model, entry in info.get("models", {}).items():
            costs = entry.get("cost", {})
            if not isinstance(costs, dict):
                continue
            rates = [number(costs.get(k)) for k in ("input", "cache_read", "output")]
            value = price(*(rate / 1_000_000 if rate is not None else None for rate in rates), "models.dev")
            if value:
                result[f"{provider}/{model}"] = value
    return result


def normalize_openrouter(data):
    result = {}
    for entry in data.get("data", []):
        model = entry.get("id")
        costs = entry.get("pricing", {})
        if not model or not isinstance(costs, dict):
            continue
        value = price(number(costs.get("prompt")), number(costs.get("input_cache_read")),
                      number(costs.get("completion")), "openrouter")
        if value:
            result[model] = value
    return result


def fetch_json(url):
    last_error = None
    for attempt in range(3):
        try:
            request = urllib.request.Request(url, headers={"User-Agent": "Wakapi-AI-Overview/1.0"})
            with urllib.request.urlopen(request, timeout=30) as response:
                return json.load(response)
        except Exception as error:
            last_error = error
            if attempt < 2:
                time.sleep(2 ** attempt)
    raise last_error


def refresh(output, aliases_file):
    parsers = {"litellm": normalize_litellm, "models.dev": normalize_models_dev,
               "openrouter": normalize_openrouter}
    aliases = json.loads(pathlib.Path(aliases_file).read_text())
    if not isinstance(aliases, dict):
        raise ValueError("aliases must be a JSON object")
    models = {}
    succeeded = []
    for source in ("litellm", "models.dev", "openrouter"):
        try:
            current = parsers[source](fetch_json(SOURCES[source]))
        except Exception as error:
            print(f"{source}: {type(error).__name__}: {error}")
            continue
        for model, value in current.items():
            models.setdefault(model, value)
        succeeded.append(source)
        print(f"{source}: {len(current)} models")
    if not succeeded:
        raise RuntimeError("all pricing catalogs failed; existing cache preserved")
    payload = {"updated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
               "sources": succeeded, "aliases": aliases, "models": models}
    output = pathlib.Path(output)
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=output.parent, prefix=".ai-pricing-", delete=False) as temp:
        path = pathlib.Path(temp.name)
        json.dump(payload, temp, separators=(",", ":"))
        temp.flush()
        os.fsync(temp.fileno())
    try:
        path.chmod(0o644)
        os.replace(path, output)
    finally:
        path.unlink(missing_ok=True)
    print(f"updated {output}: {len(models)} models, sources={','.join(succeeded)}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True)
    parser.add_argument("--aliases", required=True)
    args = parser.parse_args()
    refresh(args.output, args.aliases)
