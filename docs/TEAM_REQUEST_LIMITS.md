# Independent team request limits

Team keys share the team's balance, execution concurrency and RPM. Personal
keys keep the user's personal balance, concurrency and RPM. A team limit is a
single aggregate ceiling across all member keys, groups and providers; it is
not multiplied by the number of members.

## Administration and member view

The administrator's **Teams → Edit** dialog updates `concurrency` and
`rpm_limit`. Both accept nonnegative integers; `0` means unlimited. Team balance
continues to use the existing balance adjustment action and ledger. The team
list, details and member page display the shared limits. Members and the team
owner can read these values; only a platform administrator can change them.

`PATCH /api/v1/admin/teams/:id` accepts optional `concurrency` and `rpm_limit`
alongside `name`, `status` and `member_limit`. Omitted fields keep their values.
All supplied fields are validated before one SQL update, so an invalid value
cannot partially apply other edits. Negative values return
`TEAM_CONCURRENCY_INVALID` or `TEAM_RPM_INVALID`; non-integer JSON is rejected
by request decoding. Existing endpoint error serialization is unchanged.

```json
{"concurrency": 10, "rpm_limit": 100}
```

`GET /api/v1/team`, administrator team detail and administrator team list return
both fields. Saving a change affects later admissions for every member, including
later Responses WebSocket turns. Accepted requests retain their existing leases;
reducing the ceiling does not abort them. Increasing it admits more work
immediately. Personal edits and ownership transfers do not change team limits.
Member daily/weekly/monthly spending allowances remain separate controls over
spending the common wallet. Group access and pricing retain the existing
team-owner authorization contract.

## Runtime scope

- Team authentication keeps the actual actor and billing identity, then attaches
  the team's wallet and request limits. The payer's personal RPM override is not
  used for team requests.
- Personal RPM uses the user-global ceiling and the existing user/group override.
  Team RPM uses the team-global ceiling and the group's public default, counted
  per `(team, group)`. All applicable ceilings still apply in parallel. There is
  no member-specific RPM or concurrency policy and no per-key RPM policy.
- Internal request-limit cache IDs reserve negative team IDs. These IDs are used
  only in concurrency/RPM caches, waiting counters and lease recovery. They are
  never substituted for persisted user IDs, task actors or billing users.
- Every concurrency admission reads the saved team limit from PostgreSQL.
  This capacity read does not require current membership or a live owner: already
  accepted durable work can complete after dissolution. New requests still pass
  the existing team lifecycle authentication and remain blocked after dissolution.
  HTTP, Responses WS, Live and detached image execution use the same team scope.
  Batch image workers use their stored team ID. The existing unfinished-provider-
  task count is a separate policy and remains unchanged.
- Live records store their creation-time concurrency scope so renewal and release
  use the original pool. Pre-upgrade Live records without this field retain
  their original personal lease scope. Billing identity remains unchanged.
- RPM minute windows, rejection codes, per-request deduplication, simple-mode
  bypass and dependency-failure policies follow [gateway admission](GATEWAY_ADMISSION.md).
  Concurrency is fail-closed; existing RPM Redis-error behavior is fail-open.

## Migration and creation

Migration `334_team_request_limits.sql` adds `teams.concurrency` and
`teams.rpm_limit`. Existing teams copy the current owner's numerical settings
once, normalizing legacy negative (unlimited) values to zero. Creating a team
also normalizes and snapshots its initial owner's settings once in the
creation transaction. Subsequent edits are entirely independent; ownership
transfer does not copy the new owner's values. Personal settings, balances,
wallet entries and historical task funding are not rewritten.

Deploy with the matching backend, frontend and generated Ent/Wire files. The
migration runs through the existing forward-only runner. Drain old gateway
instances before the new gateway admits work: old binaries still share the
owner's personal limits and do not enforce the new team pool. Replacing the
binary with an old version is not a behavior-preserving rollback. Preserve the
new schema/data and use a reviewed forward repair or coordinated rollback plan.

## Verification boundary

Local validation uses an isolated PostgreSQL clone and Redis, focused static
checks and the embedded application. It covers persisted edits, initial migration,
member access, independent counters, real concurrency lease admission/release,
ownership transfer and Live scope persistence. Gateway requests deliberately
stop at model admission without supplier generation. This does not establish
paid supplier behavior, a completed live-provider handshake, or production
installation. Source, candidate acceptance, release and deployment remain
separate states.
