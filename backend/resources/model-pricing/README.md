# Model Pricing Data

This directory contains the Sub2API Plus bundled model-pricing fallback copy.

## Source
Remote refresh discovers the latest GitHub Release through a manifest:

- Manifest: `https://github.com/skyqingname/gotocc-router/releases/latest/download/model-pricing-manifest.json`
- Immutable release asset: declared by the manifest
- Upstream data source for maintainers: https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json

The sole-maintainer GitHub Release publication boundary is the source of trust.
The application accepts Release pricing only after validating the manifest
version, immutable asset URL, dedicated HTTPS host policy, response limits,
manifest-bound SHA-256 digest, and pricing JSON. No deployment key is required.

## Purpose
This local copy serves as a fallback when the remote file cannot be downloaded due to:
- Network restrictions
- Firewall rules
- DNS resolution issues
- GitHub being blocked in certain regions
- Docker container network limitations

## Update Process

The pricing service will:

1. Load a validated local cache when one exists
2. Otherwise make the bundled repository pricing immediately available
3. Automatically check the latest Release manifest before downloading data
4. Keep the current cache when a manifest, hash, URL policy, JSON, or version check fails

Remote responses are bounded before parsing:

- Manifest: 64 KiB
- Pricing data: 32 MiB

## Runtime Cache

`model_pricing.verified-cache.json` is the authoritative runtime cache. It
retains its historical filename for upgrade compatibility. It contains the
exact manifest and pricing bytes in one atomically replaced bundle. The
application validates the immutable pricing URL, digest, and JSON data again
on every startup before using it.

The following files are compatibility mirrors for operators and upgrades from
older versions:

- `model_pricing.json`
- `model_pricing.manifest.json`

A new runtime writes the authoritative bundle first, then refreshes these
mirrors. A mirror write failure does not invalidate an already committed
bundle. When no bundle exists, a valid legacy data-and-manifest cache is
validated and migrated automatically. Schema-v1 bundles containing the former
signature field remain readable and are rewritten as schema v2; obsolete
standalone `.sig` files are ignored.

## Release Update

Pricing data changes must be reviewed and published as immutable Release
assets. Do not point runtime configuration at a mutable branch. The release
workflow publishes `model-pricing.json` and `model-pricing-manifest.json` from
the same locally accepted package as the application archive.

To refresh this bundled fallback before a release, update it from the upstream
source, review the diff, and let the release workflow calculate its digest:

```bash
curl -fsS https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json -o model_prices_and_context_window.json
```

## File Format

Explicit model adaptations are maintained in the repository-root
`model-pricing-defaults.json`. Run `python3 tools/generate_model_pricing_defaults.py`
from the repository root to synchronize them into the bundled catalog;
`make test-docs` checks synchronization. These entries add catalog defaults and
do not override explicit group, channel, or configured runtime prices.

GPT-6 Astra prices were verified against the
[official model page](https://developers.openai.com/api/docs/models/gpt-6-astra)
on 2026-09-05. Per million tokens, Standard input/output/cache read/cache write
are $10/$50/$1/$12.50. Above 272K input tokens, input and cache rates double
and output uses 1.5x for the full request. Fast rates are 2x Standard;
Batch/Flex catalog rates are half Standard. Existing endpoint support and
service-tier selection determine which rate applies.

GPT-6 Sol and Luna defaults were verified against the
[official pricing page](https://developers.openai.com/api/docs/pricing) on
2026-09-23. Per million tokens, Standard input/cache read/cache write/output
are $2/$0.20/$2.50/$10 for Sol and $0.10/$0.01/$0.125/$0.50 for Luna.
Long-context, Fast, Batch, and Flex rates follow the same published tiers.
The bundled defaults replace stale remote catalog entries for these two models;
explicit group and channel prices keep their configured values.

Claude Opus 5.5 prices were verified against the
[official model page](https://platform.claude.com/docs/en/models/opus-5-5/overview)
on 2026-09-23. Per million tokens, input/output/cache read/5-minute cache
write/1-hour cache write are $4/$20/$0.20/$5/$8. Fast mode doubles each rate;
Batch halves them. The full 1M context window uses these same rates. Its
bundled default also replaces a stale remote catalog entry.

The file contains JSON data with model pricing information including:
- Model names and identifiers
- Input/output token costs
- Context window sizes
- Model capabilities
