Sub2API Plus v0.1.178+custom.004

## Highlights

- Restore Content Moderation to current direct-user text and images so a
  policy violation is attributed only to a user submission, not to platform
  or tool content.
- Keep Prompt Audit on the same canonical extraction document, including
  instructions, tool traffic, and assistant or model items, so the security
  boundary stays fully visible.
- Treat incomplete extraction as a failure for both engines before either
  selection policy is applied.
- Preserve every active GotoCC production contract (LC-001 through LC-004 and
  LC-006 through LC-012) while keeping retired LC-005 absent.

## Changed

- Select Chat and Anthropic user-role content, plus the protocol-defined
  roleless user forms in Responses, Live, and Gemini, as Content Moderation
  inputs. Direct Alpha Search queries, embedding strings, and media prompts
  remain eligible.
- Exclude instructions, system or developer context, reusable prompt
  variables, assistant or model messages, reasoning, tool definitions,
  calls, results, approval responses, and tool-produced images from Content
  Moderation while leaving those segments available to Prompt Audit.
- Keep the official v0.1.178 baseline and Plus customizations; this
  iteration does not change the embedded Codex identity precedence.
- Compose asynchronous-image failed-task deletion and private exact ZIP keys
  with durable PostgreSQL ownership, same-user URL renewal, and streaming
  uploads; storage keys are not exposed in public JSON.
- Preserve atomic registration-time consumption for both one-time and reusable
  invitation codes.
- Keep all NewAPI-backed video models in one OpenAI-compatible group. Accept the
  legacy `/videos/generations` create, status, and content aliases used by
  Infinite Canvas and normalize them to NewAPI's canonical `/v1/videos` task
  surface without introducing Composite or model-prefix routing. Channel
  create/update validation now also accepts the existing `video` billing mode,
  allowing per-second rules to be maintained through the normal admin API.

## Fixed

- Restore the `v0.1.177+custom.003` user-attribution rule that a later
  shared-extractor expansion had broadened beyond direct-user content.
- Satisfy audit-content lint after the extraction-scope change.

## Compatibility and migration

- Existing data remains compatible. Startup applies migrations 224, 225, 226,
  and 228 in lexical order; no manual migration command is required. Migration
  224 normalizes the Codex fingerprint mode for top-level OpenAI OAuth
  accounts (defaulting missing or invalid values to `device`) and removes that
  field from non-applicable accounts. Migrations 225, 226, and 228 add channel
  time pricing, monitor quota modes, and the expanded platform-quota constraint;
  migration 227 is intentionally unused.
- Migrations are forward-only. Rolling back the application does not undo the
  migration or its database trigger; back up PostgreSQL before upgrading. A
  database rollback requires restoring a backup or applying an audited
  compensating SQL migration.
- Existing GotoCC migrations `224_add_image_objects.sql` and
  `225_restore_openai_video_prices.sql` remain immutable. Their full filenames
  differ from the new Plus `224` and `225` files, so both lineages remain
  independently tracked.

## Known issues

- None known.

## Upstream baseline

Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
