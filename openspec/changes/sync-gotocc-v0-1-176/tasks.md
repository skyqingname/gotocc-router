## 1. Source Integration

- [x] 1.1 Freeze and fully verify the deployed dirty source baseline.
- [x] 1.2 Fetch and verify Plus tag `v0.1.176+custom.001`.
- [x] 1.3 Create the isolated candidate directly from the peeled tag commit.
- [x] 1.4 Apply GotoCC semantic source without old generated output.
- [x] 1.5 Combine upstream API-key validation with team scope resolution.
- [x] 1.6 Regenerate Ent and Wire.

## 2. Release Metadata

- [x] 2.1 Set all version declarations to `0.1.176+custom.002`.
- [x] 2.2 Record official and Plus input identities and migration impact.
- [x] 2.3 Commit a clean, reviewable candidate.

## 3. Verification

- [x] 3.1 Pass markers.
- [x] 3.2 Pass targeted tests.
- [x] 3.3 Pass full tests and public-page browser checks. Authenticated clone
  rehearsal remains a separate approval gate.
- [x] 3.4 Pass release checks and create a Linux/amd64 `NOT DEPLOYED` artifact.
- [x] 3.5 Record generated fingerprints and final artifact SHA-256.
