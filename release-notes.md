GoToCC 0.2.1+custom.003

## Changes

- Adapted Plus v0.2.1+custom.002 (1b95c72f186275f582714bed38a1a4af122429f1), based on official Sub2API v0.2.1 (578785ee7fb35030b094b69624efe25670a36f5f).
- Disconnect streaks now use independent client sessions. The administrator event panel adds user, key, request/session ID, protocol, time and usage-source filters.
- Retained upstream independent transport/originator recognition, official Codex search profiles and OAuth-only group compatibility, preserving credential identity precedence.
- Included upstream Grok audio cancellation handling and gRPC 1.83.1.
- Preserved every active GoToCC contract: invitations, teams, async video settlement, permanent image objects, Model Plaza and branding, owned updates, smart keys, actual HTTP status and downstream-disconnect attribution.
- Preserved audit templates/scoring, per-node timeouts, text preview, actual-model probes and credential diagnostics, and Astra defaults for one-click access/CCS.

## Migration, configuration and rollback

Upgrading from owned 0.2.1+custom.002 adds only migration 260. Upstream 256_client_disconnect_session_scope.sql is imported under this new number with identical SQL contents; owned migrations through 259 are unchanged.

The migration backfills disconnect-event user email/key names and legacy session scope, replaces the event primary key, rebuilds derived risk state per session, and creates four ordinary indexes. Existing events remain; derived streak counters restart. ALTER TABLE, primary-key replacement and non-concurrent indexes take table locks. Backfill/index creation requires additional disk and WAL capacity. Stop the old core before startup and retain matched database/runtime backups; never overlap writer versions.

Consecutive-disconnect banning is turned off and its generation is advanced. Administrators must review session scope before deciding to re-enable it. No new environment variables or Redis flush are required. Balances, usage, credentials and image/video task ownership are preserved.

The old application is incompatible with the new risk-state schema. Binary-only rollback is unsupported. After migration or new business writes, preserve current data and use a forward fix or separately planned schema adaptation; never overwrite it with an old dump. Upgrades from 0.2.0+custom.004 also apply the previously released migrations 253-259 for smart keys, original batch groups, upstream request IDs/concurrent index, Astra monitor configuration, max-reasoning multiplier and Codex manifest; see docs/GOTOCC_PLUS_MIGRATION.md.

## Local acceptance

The Linux/amd64 final archive and runtime pricing resources are built locally and the same package runs the preview. Await user manual acceptance before publication; online updates use the owned version panel. No paid image/video probes are created.
