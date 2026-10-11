# LM Fingerpoint Detector

Source: https://github.com/Ikaleio/lm-detector

Revision: 97eb41f78722d32524265bb259fe6258856ca470

MIT license, copyright (c) 2026 xqy2006. The full license is in the product root
`LICENSE.lm-detector`, included in release assets by the existing LICENSE pattern.

`challenge-browser.js`, `fingerprint-core.js`, `positional.ts` and the numerical
ranking/calibration in `shared-detector.ts` are from this revision. Adaptations:
standalone type declarations; a paired-bank mismatch reports an error instead
of invoking the legacy ranker; `bank.json` retains only the metadata used by the
v2 ranker. `detector.json.gz` contains the original trained artifact, serialized
compactly and gzip-compressed without altering its numbers or calibration.

The trained numerical constants are part of the frozen reference protocol, not
site settings. Site execution parameters come from `model-fingerprint.json`.
