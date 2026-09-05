#!/usr/bin/env python3
"""Sync explicitly maintained model prices into the bundled pricing catalog."""
import argparse
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    models = json.loads((ROOT / "model-pricing-defaults.json").read_text())["models"]
    target = ROOT / "backend/resources/model-pricing/model_prices_and_context_window.json"
    source = target.read_text()
    catalog = json.loads(source)
    if args.check:
        for name, entry in models.items():
            if catalog.get(name) != entry:
                raise SystemExit(f"bundled model pricing is out of sync: {name}")
        return
    for name, entry in models.items():
        rendered = json.dumps(entry, ensure_ascii=False, indent=2).replace("\n", "\n  ")
        match = re.search(r"^  " + re.escape(json.dumps(name)) + r":\s*", source, re.MULTILINE)
        if match:
            _, length = json.JSONDecoder().raw_decode(source[match.end():])
            source = source[:match.end()] + rendered + source[match.end() + length:]
        else:
            source = source.rstrip()[:-1].rstrip() + ",\n  " + json.dumps(name) + ": " + rendered + "\n}\n"
    target.write_text(source)


if __name__ == "__main__":
    main()
