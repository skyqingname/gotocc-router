# 0.2.15+custom.007 — Local async image storage

New asynchronous image results can be stored on the server filesystem through an explicit root JSON configuration. Existing S3 images remain readable through their original storage; task submission, ownership checks, URL renewal, ZIP downloads and billing keep their existing contracts.

Local images use expiring signed URLs served by Plus. Back up their persistent directory together with the database and configuration. No database migration is required; older S3-only binaries cannot read newly created local object references.

Upstream baseline remains Sub2API v0.2.15 with the existing Plus customizations. This candidate has not been published to GitHub. Runtime validation is recorded in the Oracle maintenance workspace.
