#!/usr/bin/env python3
"""Download models.dev and write the trimmed model catalog bundled with each release.

The source URL and the providers to keep, in lookup priority order, are declared in
model-plaza-catalog.json. Only the fields the model plaza shows are kept.
"""
import datetime
import json
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TARGET = ROOT / "backend/internal/pkg/modelcatalog/catalog_gen.json"
MODEL_FIELDS = (
    "id", "name", "description", "family", "reasoning", "tool_call", "structured_output",
    "attachment", "open_weights", "knowledge", "release_date", "modalities", "limit",
)
COST_FIELDS = ("input", "output", "cache_read", "cache_write")


def trim_cost(cost):
    out = {key: cost[key] for key in COST_FIELDS if key in cost}
    tiers = [
        {**{key: tier[key] for key in COST_FIELDS if key in tier}, "context_over": tier["tier"]["size"]}
        for tier in cost.get("tiers", [])
        if tier.get("tier", {}).get("type") == "context"
    ]
    if tiers:
        out["tiers"] = tiers
    return out


def trim_model(model):
    out = {key: model[key] for key in MODEL_FIELDS if key in model}
    if "cost" in model:
        out["cost"] = trim_cost(model["cost"])
    return out


def main():
    config = json.loads((ROOT / "model-plaza-catalog.json").read_text())
    # models.dev 拒绝 Python 默认的 User-Agent，请求需带上明确的客户端标识。
    request = urllib.request.Request(config["source"], headers={"User-Agent": "gotocc-model-catalog"})
    with urllib.request.urlopen(request, timeout=120) as response:
        registry = json.load(response)
    providers = []
    for provider_id in config["providers"]:
        provider = registry[provider_id]
        models = [trim_model(provider["models"][key]) for key in sorted(provider["models"])]
        providers.append({"id": provider_id, "name": provider["name"], "models": models})
    snapshot = {
        "source": config["source"],
        "fetched_at": datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat(),
        "providers": providers,
    }
    TARGET.write_text(json.dumps(snapshot, ensure_ascii=False, indent=1) + "\n")
    print(f"wrote {TARGET.relative_to(ROOT)}: {sum(len(p['models']) for p in providers)} models from {len(providers)} providers")


if __name__ == "__main__":
    main()
