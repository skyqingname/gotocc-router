## 1. Candidate integration

- [x] 1.1 Verify the Plus tag, peeled commit, official baseline, and release
      metadata discrepancy.
- [x] 1.2 Create the isolated candidate from the immutable Plus tag.
- [x] 1.3 Reapply every active GotoCC contract and keep LC-005 retired.
- [x] 1.4 Resolve scheduler, Images, video, model-plaza, frontend, and generated
      provider composition conflicts.
- [x] 1.5 Regenerate Ent, Wire, and the embedded frontend.

## 2. Metadata and validation

- [x] 2.1 Set candidate version `0.1.183+custom.003` and synchronize release
      documentation.
- [x] 2.2 Prove existing migration bytes are unchanged and document migrations
      229-233 plus rollback limits.
- [x] 2.3 Pass markers and targeted gates.
- [x] 2.4 Pass the final release gate and record the Linux/amd64 binary,
      runtime-resource, and generated-output SHA-256 values.

## 3. Production authorization

- [ ] 3.1 Wait for explicit deployment authorization before any public push,
      vircs upload, backup, stop/start, migration, or core replacement.
