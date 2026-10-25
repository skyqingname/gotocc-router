# LM Fingerpoint Detector

Source: https://github.com/Ikaleio/lm-detector

Revision: 4dcb4216151cbd5066d515297720130a9fd3b198

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

The 2026-10-25 bank contains 63 identities, adding Kimi K3.1 Pro, Gemini 3.5
Flash Lite and MiniMax M3.1 Flash. The paired sequence-shape rule rejects a
reply only when both signals exceed their upstream limits. No SDK, automatic
sampling or tokenizer probe is enabled by this reference refresh.
