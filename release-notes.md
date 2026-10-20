# 0.2.17+custom.003 — Clear review outcomes in risk control

Risk-control records now say “Reviewed” with their result, or “Not reviewed” with the reason. Records without a score show “Not scored”. The input detail dialog and the unreviewed filter follow the same wording. The page no longer presents extraction counters as review results; internal extraction diagnostics remain available.

New unreviewed records distinguish timeouts, canceled requests and unavailable review endpoints. Historical generic errors remain unreviewed without guessing their cause. Audit input selection, sampling, thresholds, blocking, penalties and production configuration are unchanged. No database migration is added.

Continues GoToCC `0.2.17+custom.002` on Plus `v0.2.17+custom.001` / official Sub2API `v0.2.17`. Both Linux/arm64 and Linux/amd64 packages share the same source and embedded frontend. This is a local candidate pending owner acceptance; publication and production installation are separate.
