#!/usr/bin/env python3
"""Embed the repository's content-audit Jev parameters in the backend."""
import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    config = json.loads((ROOT / "content-audit-jev.json").read_text())
    output = json.dumps(config, ensure_ascii=False, indent=2) + "\n"
    destination = ROOT / "backend/internal/service/content_audit_jev_gen.json"
    if args.check:
        if destination.read_text() != output:
            raise SystemExit("content audit Jev parameters are out of sync")
    else:
        destination.write_text(output)


if __name__ == "__main__":
    main()
