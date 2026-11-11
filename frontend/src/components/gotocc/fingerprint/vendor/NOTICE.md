# LM Fingerpoint Detector

Source: https://github.com/Ikaleio/lm-detector

Revision: a7e9d3056e019827904dd50722c20b4a0c896738

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

The 2026-11-11 bank contains 69 identities, adding GPT-6.1 Terra, OpenPangu 2.0
Flash/Pro, Muse Ember, Claude Fable 5.2 and Gemini 4 Carbon. Gemini 4 Argon
reference outputs are replaced with the upstream's Vertex US samples. The
numerical algorithms and sequence-shape rule are unchanged. No SDK, automatic
sampling or tokenizer probe is enabled by this reference refresh.

Known upstream-data limitation: some saved OpenPangu 2.0 Flash output groups
trigger the unchanged sequence-shape rejection. Enrollment is not a guarantee
of reliable identification; this refresh preserves the upstream thresholds and
reference artifact rather than adjusting them to make saved samples pass.
