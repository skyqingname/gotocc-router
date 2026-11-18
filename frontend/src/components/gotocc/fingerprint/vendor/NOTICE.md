# LM Fingerpoint Detector

Source: https://github.com/Ikaleio/lm-detector

Revision: 33fb43f26670be0ed7817ab13e5c4c4731ccb76a

MIT license, copyright (c) 2026 xqy2006. The full license is in the product root
`LICENSE.lm-detector`, included in release assets by the existing LICENSE pattern.

`challenge-browser.js`, `fingerprint-core.js`, `positional.ts`, `detector-codec.ts` and the numerical
ranking, calibration and sequence-shape check in `shared-detector.ts` are from this revision. Adaptations:
standalone type declarations; `bank.json` retains only the metadata used by the v2 ranker.
`detector.bin.gz` is the original `data/shared_detector.bin` float32 artifact, gzip-compressed
without altering its numbers or calibration, and decoded with the upstream `decodeDetector`.
Source modules also carry the full MIT notice as preserved browser-bundle comments.

The trained numerical constants are part of the frozen reference protocol, not
site settings. Site execution parameters come from `model-fingerprint.json`.

The 2026-11-18 bank contains 72 identities, adding GPT-6.2 Sol and GPT-6.2 Astra
from OpenRouter on top of the 2026-11-15 bank; the detector artifact was retrained upstream
with refreshed Opus 4.7 and 4.8 references. This revision also takes the upstream sequence-shape
fix (#40): a round is rejected only when lag-1 autocorrelation, the adjacent small-step share and
the share of close pairs up to three positions apart all exceed their limits (-0.23, 5%, 7%).
The ranking, calibration and positional decision are otherwise unchanged.
No SDK, automatic sampling or tokenizer probe is enabled by this reference refresh.

Known upstream-data limitation: some saved OpenPangu 2.0 Flash output groups
trigger the unchanged sequence-shape rejection. Enrollment is not a guarantee
of reliable identification; this refresh preserves the upstream thresholds and
reference artifact rather than adjusting them to make saved samples pass.
