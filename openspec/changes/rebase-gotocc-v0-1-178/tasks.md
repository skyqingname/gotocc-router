## 1. Candidate integration

- [x] 1.1 Verify Plus `v0.1.178+custom.001` tag, peeled commit, and archive SHA.
- [x] 1.2 Create the isolated candidate from the immutable tag.
- [x] 1.3 Reapply active GotoCC contracts and keep LC-005 retired.
- [x] 1.4 Resolve invitation, asynchronous-image, route, middleware, and test
  composition conflicts.
- [x] 1.5 Regenerate Ent and Wire from semantic schemas and providers.
- [x] 1.6 Regenerate and verify the embedded frontend.

## 2. Metadata and validation

- [x] 2.1 Set candidate version `0.1.178+custom.002`.
- [x] 2.2 Record immutable baselines, migration impact, and rollback limits.
- [x] 2.3 Pass markers, targeted, and full gates.
- [x] 2.4 Pass the release gate and record Linux/amd64 binary SHA-256,
  runtime resource SHA-256, and
  generated artifact fingerprints.

## 3. Production authorization

- [x] 3.1 Report the four production migrations, including account data
  backfill and binary-only rollback limits.
- [ ] 3.2 Wait for explicit production deployment authorization before any
  vircs upload, backup, stop/start, migration, or replacement.
