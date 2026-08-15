## 1. Baseline and specification

- [x] 1.1 Create the candidate directly from the immutable Plus release tag.
- [x] 1.2 Preserve and verify Plus license, notice, upstream mapping, dependency
  files, and complete tracked source tree.
- [x] 1.3 Record immutable LC numbering, LC-005 retirement, and GotoCC homepage
  adaptation ownership.
- [x] 1.4 Validate this OpenSpec change strictly.

## 2. Persistent-data and cache compatibility

- [x] 2.1 Add fail-closed migration-lineage preflight before any migration can
  execute.
- [x] 2.2 Add the fourteen reviewed filename/checksum/schema equivalence rules
  and database-free coverage for match, mismatch, partial, and idempotent cases.
- [x] 2.3 Add production-only migration ownership checks for reusable
  invitations, teams, and durable image objects.
- [x] 2.4 Add old-to-new JWT, refresh-token, Redis key, and JSON compatibility
  fixtures without connecting to production Redis.
- [x] 2.5 Document prefix-specific cache rebuild and production drain rules.
- [x] 2.6 Keep Plus 218 immutable and restore OpenAI/Jimeng video prices with
  compensating migration 225.

## 3. GotoCC backend contracts

- [x] 3.1 Reimplement reusable invitation code storage, registration/OAuth
  consumption, administration API, Wire injection, and double-path tests.
- [x] 3.2 Reimplement team lifecycle, team keys, actor/billing-owner/team
  attribution, member allowances, invitation limits, ownership transfer, and
  billing tests.
- [x] 3.3 Integrate durable image object ownership and URL renewal with Plus
  async task history and existing Redis task state.
- [x] 3.4 Integrate Jingmeng video routes beside Plus Grok routes.
- [x] 3.5 Integrate Images model passthrough and Gemini response normalization
  without altering non-Images mapping.
- [x] 3.6 Add lightweight public homepage statistics and `/models` Model Plaza
  compatibility without restoring the retired marketplace.

## 4. GotoCC frontend contracts

- [x] 4.1 Add reusable invitation administration to the Plus navigation,
  routing, API, locale, and views.
- [x] 4.2 Add team user/admin workflows and team-aware key and usage UI.
- [x] 4.3 Add one-click key import to the Plus key-management interactions.
- [x] 4.4 Adapt the GotoCC homepage to Plus components, routing, state, API,
  locale, and build structure while retaining local branding and `logo.png`.
- [x] 4.5 Keep Plus Model Plaza as the only marketplace and add only the
  approved compatibility entry.

## 5. Generation and verification

- [x] 5.1 Regenerate Ent and Wire from the candidate semantic sources.
- [x] 5.2 Regenerate the embedded frontend dist from the candidate frontend.
- [x] 5.3 Run Plus formatting, lint, backend, migration, frontend, and embedded
  tests.
- [x] 5.4 Run GotoCC markers, targeted, full, and release verification with no
  active LC gaps and LC-005 still absent.
- [x] 5.5 Run desktop and 390px homepage, team, key, invitation, image, and
  Model Plaza browser checks.
- [x] 5.6 Record source commit, generated artifact hashes, test evidence, and
  remaining environment-only limits.

## 6. Separately authorized operations

- [ ] 6.1 Rehearse against isolated PostgreSQL and Redis production-consistent
  copies and measure migrations, locks, WAL, data counts, and session behavior.
- [ ] 6.2 Build and restore-test current production binary/image, PostgreSQL,
  Redis, compose, and configuration rollback assets.
- [ ] 6.3 Obtain separate authorization before production database writes,
  Redis mutation, `.env` changes, upload, deployment, restart, or Git push.
