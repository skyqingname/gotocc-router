## 1. Candidate Integration

- [x] 1.1 Fetch and verify Plus `v0.1.177+custom.001`.
- [x] 1.2 Create an isolated candidate from the immutable tag.
- [x] 1.3 Reapply the GotoCC production contracts and resolve usage-log field
  composition.
- [x] 1.4 Bind the historical same-prefix migrations to exact path and SHA.
- [ ] 1.5 Regenerate Ent, Wire, and embedded frontend output.
- [x] 1.6 Repair team email-link fallback with api_base_url, trusted-origin
  validation, and focused invite/reissue/ownership coverage.

## 2. Metadata and Validation

- [x] 2.1 Set candidate version `0.1.177+custom.002`.
- [x] 2.2 Update release metadata and notes without publication.
- [ ] 2.3 Pass markers, targeted, full, and release gates.
- [ ] 2.4 Build and record Linux/amd64 package SHA-256 and generated hashes.

## 3. Production confirmation (not part of the local hook)

- [ ] 3.1 After `markers → targeted → full → release`, report schema impact and
  rollback limits. Do not invent an isolation rehearsal as a hook step.
- [ ] 3.2 Wait for an explicit production-deploy confirmation before touching
  vircs binaries, databases, or configuration.
