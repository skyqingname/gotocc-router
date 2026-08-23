## ADDED Requirements

### Requirement: Accounts may define multiple usage-alert rules

The system SHALL allow an administrator to store up to 20 independently
enabled usage-alert rules for one upstream account in
`accounts.extra.usage_alert_rules`. Each rule MUST have a stable identifier,
one supported delivery channel, an HTTPS webhook URL, a five-field report cron,
an optional force-probe setting, and its own run metadata. Saving the new rule
set MUST clear the legacy single-rule WeCom fields, while reading an account
that only has those legacy fields MUST expose an equivalent WeCom rule.

#### Scenario: Administrator saves two schedules for one account

- **WHEN** an administrator saves two valid usage-alert rules with distinct IDs
- **THEN** both rules MUST be persisted for the account
- **THEN** each rule MUST retain independent enabled state, schedule, and run metadata

#### Scenario: Legacy WeCom configuration is read

- **WHEN** an account has no `usage_alert_rules` array but has a valid legacy
  WeCom usage-alert configuration
- **THEN** the admin API MUST return an equivalent single WeCom rule
- **THEN** the legacy webhook and schedule MUST remain usable until the rule set is saved

### Requirement: Usage-alert administration must preserve compatibility

The system SHALL expose GET, PUT, and test-send operations under
`/admin/accounts/{id}/usage-alert`. The legacy
`/admin/accounts/{id}/wecom-usage-alert` paths MUST remain aliases for the same
operations. Admin responses SHALL return complete webhook URLs because the
administrator must be able to inspect and edit the configured destination.

#### Scenario: Legacy client loads alert configuration

- **WHEN** an authorized administrator calls the legacy WeCom usage-alert GET path
- **THEN** the request MUST execute the same configuration read as the new usage-alert path

#### Scenario: Administrator tests an unsaved rule

- **WHEN** an authorized administrator submits a valid draft rule to the test-send operation
- **THEN** the system MUST fetch current usage and send a report to that draft rule's webhook
- **THEN** the test MUST ignore the threshold gate without persisting the draft

### Requirement: Alert destinations and schedules must be validated

The system SHALL accept only `wecom`, `feishu`, and `custom` channels. WeCom
and Feishu destinations MUST use HTTPS and their approved bot webhook host and
path. Custom destinations MUST use HTTPS. Enabled rules MUST provide a valid
five-field report cron. Threshold-enabled rules MUST additionally provide a
threshold from 1 through 99, a valid five-field watch cron, and a positive
cooldown no longer than 30 days. Quiet-hour ranges MUST use normalized daily
time ranges.

#### Scenario: Invalid WeCom host is rejected

- **WHEN** an administrator saves a WeCom rule whose webhook uses an unrelated host
- **THEN** the update MUST be rejected without replacing the stored rules

#### Scenario: Threshold rule lacks a watch schedule

- **WHEN** an administrator enables a threshold for a rule without a valid
  `threshold_watch_cron`
- **THEN** the update MUST be rejected without scheduling the rule

### Requirement: Scheduled reports and threshold alerts must run independently

The system SHALL check due usage-alert rules once per minute. A due report
schedule MUST send a usage-window report and advance its next-run metadata even
when delivery fails. A due threshold watch MUST fetch usage, send only when the
maximum window utilization reaches the configured threshold, and enforce the
configured cooldown after a successful threshold alert. Report and threshold
schedules MUST skip configured quiet hours and advance to the next eligible
time.

#### Scenario: Due report delivery fails

- **WHEN** an enabled rule's report schedule is due and its webhook delivery fails
- **THEN** the rule MUST record the delivery error and last-run time
- **THEN** the next report run MUST still be advanced

#### Scenario: Utilization remains below threshold

- **WHEN** a threshold watch is due and every usage window is below the configured percentage
- **THEN** no threshold alert MUST be sent
- **THEN** the next threshold watch MUST be scheduled normally

#### Scenario: Threshold is reached during cooldown

- **WHEN** utilization reaches the configured percentage before the rule's cooldown ends
- **THEN** the system MUST NOT fetch usage or send another threshold alert
- **THEN** the next threshold check MUST be deferred until the cooldown ends
