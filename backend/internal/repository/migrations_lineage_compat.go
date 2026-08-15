package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

type legacyMigrationLineageRule struct {
	legacyFilename string
	targetFilename string
	legacyChecksum string
	targetChecksum string
	schemaCheckSQL string
}

// These rules are limited to the reviewed GotoCC production lineage. They do
// not relax normal checksum validation and must never become a generic alias
// mechanism for unrelated migrations.
var legacyMigrationLineageRules = []legacyMigrationLineageRule{
	{
		legacyFilename: "187_add_usage_log_session_id.sql",
		targetFilename: "189_add_usage_log_session_id.sql",
		legacyChecksum: "3f6a27f01ce117cb981557d2bea847faa1893af8d147772b0159b3a0558181ec",
		targetChecksum: "3f6a27f01ce117cb981557d2bea847faa1893af8d147772b0159b3a0558181ec",
		schemaCheckSQL: `SELECT COUNT(*) = 2
FROM information_schema.columns
WHERE table_schema = 'public'
  AND data_type = 'character varying'
  AND character_maximum_length = 255
  AND is_nullable = 'YES'
  AND (table_name, column_name) IN (
    ('usage_logs', 'session_id'),
    ('batch_image_jobs', 'session_id')
  )`,
	},
	{
		legacyFilename: "188_allow_live_usage_request_type.sql",
		targetFilename: "190_allow_live_usage_request_type.sql",
		legacyChecksum: "0233dba07a75bd9c740402a64e3af75c2a3884dfc8c4b63145df115e716fd35e",
		targetChecksum: "0233dba07a75bd9c740402a64e3af75c2a3884dfc8c4b63145df115e716fd35e",
		schemaCheckSQL: `SELECT COUNT(*) = 1
FROM pg_constraint c
JOIN pg_class t ON t.oid = c.conrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE n.nspname = 'public'
  AND t.relname = 'usage_logs'
  AND c.conname = 'usage_logs_request_type_check'
  AND c.contype = 'c'
  AND c.convalidated
  AND pg_get_constraintdef(c.oid) LIKE '%request_type >= 0%'
  AND pg_get_constraintdef(c.oid) LIKE '%request_type <= 5%'`,
	},
	{
		legacyFilename: "189_add_group_allow_live.sql",
		targetFilename: "191_add_group_allow_live.sql",
		legacyChecksum: "51172b10c160e7f560346dbaf736dc8e92feb793cd00169f5fb876c399460862",
		targetChecksum: "51172b10c160e7f560346dbaf736dc8e92feb793cd00169f5fb876c399460862",
		schemaCheckSQL: `SELECT COUNT(*) = 1
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'groups'
  AND column_name = 'allow_live'
  AND data_type = 'boolean'
  AND is_nullable = 'NO'
  AND column_default = 'false'`,
	},
	{
		legacyFilename: "190_add_users_email_alias_dedup_index_notx.sql",
		targetFilename: "192_add_users_email_alias_dedup_index_notx.sql",
		legacyChecksum: "14f65700704f92ca397d407bba98b5719c38503a7085a3253fabfc08b1c2dac5",
		targetChecksum: "14f65700704f92ca397d407bba98b5719c38503a7085a3253fabfc08b1c2dac5",
		schemaCheckSQL: `SELECT COUNT(*) = 1
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_class t ON t.oid = i.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE n.nspname = 'public'
  AND t.relname = 'users'
  AND idx.relname = 'idx_users_email_dot_stripped'
  AND i.indisvalid
  AND i.indisready
  AND pg_get_indexdef(i.indexrelid) LIKE '%replace(lower(TRIM(BOTH FROM email)), ''.''::text, ''''::text)%'
  AND pg_get_indexdef(i.indexrelid) LIKE '%text_pattern_ops%'
  AND pg_get_indexdef(i.indexrelid) LIKE '%deleted_at IS NULL%'`,
	},
	{
		legacyFilename: "194_passkey_credentials.sql",
		targetFilename: "196_passkey_credentials.sql",
		legacyChecksum: "d79e7093f28b1a2ba923da35d7376423683c9a5d21dffd0e581f3c45b5afd817",
		targetChecksum: "d79e7093f28b1a2ba923da35d7376423683c9a5d21dffd0e581f3c45b5afd817",
		schemaCheckSQL: `SELECT
  to_regclass('public.passkey_user_handles') IS NOT NULL
  AND to_regclass('public.passkey_credentials') IS NOT NULL
  AND to_regclass('public.passkey_credentials_user_id_idx') IS NOT NULL
  AND to_regclass('public.passkey_credentials_last_used_at_idx') IS NOT NULL
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'passkey_user_handles'
         AND column_name IN ('user_id', 'user_handle', 'created_at')) = 3
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'passkey_credentials'
         AND column_name IN ('id', 'user_id', 'credential_id', 'name',
                             'credential_data', 'last_used_at', 'created_at', 'updated_at')) = 8
  AND (SELECT COUNT(*) FROM pg_constraint c
       JOIN pg_class t ON t.oid = c.conrelid
       JOIN pg_namespace n ON n.oid = t.relnamespace
       WHERE n.nspname = 'public'
         AND t.relname IN ('passkey_user_handles', 'passkey_credentials')
         AND c.contype = 'f' AND c.confdeltype = 'c') = 2`,
	},
	{
		legacyFilename: "192_group_profit_control.sql",
		targetFilename: "198_group_profit_control.sql",
		legacyChecksum: "c128560346272041ce20dbed1d54955b7be95ca614bc1cd821a1b3b0adc37063",
		targetChecksum: "c128560346272041ce20dbed1d54955b7be95ca614bc1cd821a1b3b0adc37063",
		schemaCheckSQL: `SELECT
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'groups'
     AND column_name = 'profit_control_enabled'
     AND data_type = 'boolean' AND is_nullable = 'NO'
     AND column_default = 'false') = 1
  AND
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'groups'
     AND column_name IN ('profit_min_margin', 'profit_safety_buffer')
     AND data_type = 'numeric' AND numeric_precision = 10 AND numeric_scale = 4
     AND is_nullable = 'NO' AND column_default = '0') = 2`,
	},
	{
		legacyFilename: "193_group_profit_control_auth_cache_invalidation.sql",
		targetFilename: "199_group_profit_control_auth_cache_invalidation.sql",
		legacyChecksum: "5ab25a32239e64c6a6318ed71b72f1456b9e6fc6d4c06d162601af741cb16dd5",
		targetChecksum: "5ab25a32239e64c6a6318ed71b72f1456b9e6fc6d4c06d162601af741cb16dd5",
		schemaCheckSQL: `SELECT COUNT(*) = 1
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
  AND p.proname = 'enqueue_group_auth_cache_invalidation'
  AND pg_get_functiondef(p.oid) LIKE '%profit_control_enabled%'
  AND pg_get_functiondef(p.oid) LIKE '%profit_min_margin%'
  AND pg_get_functiondef(p.oid) LIKE '%profit_safety_buffer%'
  AND EXISTS (
    SELECT 1
    FROM pg_trigger trg
    JOIN pg_class t ON t.oid = trg.tgrelid
    JOIN pg_namespace tn ON tn.oid = t.relnamespace
    WHERE tn.nspname = 'public'
      AND t.relname = 'groups'
      AND trg.tgname = 'trg_groups_auth_cache_invalidation'
      AND NOT trg.tgisinternal
      AND trg.tgenabled <> 'D'
  )`,
	},
	{
		legacyFilename: "194_add_usage_log_upstream_response_model.sql",
		targetFilename: "200_add_usage_log_upstream_response_model.sql",
		legacyChecksum: "cad520cbfcf7af7ea9acae92e5bcbe27501fd9e3ad5b02e306f4f97be4410a82",
		targetChecksum: "cad520cbfcf7af7ea9acae92e5bcbe27501fd9e3ad5b02e306f4f97be4410a82",
		schemaCheckSQL: `SELECT
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'usage_logs'
     AND column_name = 'upstream_response_model'
     AND data_type = 'character varying'
     AND character_maximum_length = 200
     AND is_nullable = 'YES') = 1
  AND
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'public' AND table_name = 'usage_logs'
     AND column_name = 'upstream_model_mismatch'
     AND data_type = 'boolean' AND is_nullable = 'YES') = 1`,
	},
	{
		legacyFilename: "195_add_usage_log_upstream_model_mismatch_index_notx.sql",
		targetFilename: "201_add_usage_log_upstream_model_mismatch_index_notx.sql",
		legacyChecksum: "692f2a75f0c62670b4d68986912bf24eb92f6377ec904d3806ff7d62b0da8355",
		targetChecksum: "692f2a75f0c62670b4d68986912bf24eb92f6377ec904d3806ff7d62b0da8355",
		schemaCheckSQL: `SELECT COUNT(*) = 1
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_class t ON t.oid = i.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE n.nspname = 'public'
  AND t.relname = 'usage_logs'
  AND idx.relname = 'idx_usage_logs_upstream_model_mismatch_created_at'
  AND i.indisvalid
  AND i.indisready
  AND pg_get_indexdef(i.indexrelid) LIKE '%created_at DESC%'
  AND pg_get_indexdef(i.indexrelid) LIKE '%id DESC%'
	  AND pg_get_indexdef(i.indexrelid) LIKE '%upstream_model_mismatch IS TRUE%'`,
	},
	{
		legacyFilename: "154_reusable_invitation_codes.sql",
		targetFilename: "220_reusable_invitation_codes.sql",
		legacyChecksum: "c1c6c6d985ce940a9e4feed68179f67a3d9e216ef2a54f02e3804c3859ad1bc3",
		targetChecksum: "c1c6c6d985ce940a9e4feed68179f67a3d9e216ef2a54f02e3804c3859ad1bc3",
		schemaCheckSQL: `SELECT
  to_regclass('public.reusable_invitation_codes') IS NOT NULL
  AND to_regclass('public.reusable_invitation_code_uses') IS NOT NULL
  AND to_regclass('public.reusable_invitation_codes_status_idx') IS NOT NULL
  AND to_regclass('public.reusable_invitation_codes_expires_at_idx') IS NOT NULL
  AND to_regclass('public.reusable_invitation_code_uses_code_id_used_at_idx') IS NOT NULL
  AND to_regclass('public.reusable_invitation_code_uses_user_id_idx') IS NOT NULL
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'reusable_invitation_codes'
         AND column_name IN ('id', 'code', 'status', 'max_uses', 'used_count',
                             'expires_at', 'notes', 'created_at', 'updated_at')) = 9
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'reusable_invitation_code_uses'
         AND column_name IN ('id', 'code_id', 'user_id', 'email', 'auth_source', 'used_at')) = 6
  AND (SELECT COUNT(*) FROM pg_constraint c
       JOIN pg_class t ON t.oid = c.conrelid
       JOIN pg_namespace n ON n.oid = t.relnamespace
       WHERE n.nspname = 'public'
         AND t.relname = 'reusable_invitation_codes'
         AND c.conname IN (
           'reusable_invitation_codes_pkey',
           'reusable_invitation_codes_code_key',
           'reusable_invitation_codes_status_check',
           'reusable_invitation_codes_max_uses_check',
           'reusable_invitation_codes_used_count_check'
         )) = 5
  AND (SELECT COUNT(*) FROM pg_constraint c
       JOIN pg_class t ON t.oid = c.conrelid
       JOIN pg_namespace n ON n.oid = t.relnamespace
       WHERE n.nspname = 'public'
         AND t.relname = 'reusable_invitation_code_uses'
         AND c.contype = 'f'
         AND c.confdeltype = 'a') = 2`,
	},
	{
		legacyFilename: "191_add_teams.sql",
		targetFilename: "221_add_teams.sql",
		legacyChecksum: "11f5a4052b546f70efe4699259c8276afe421599f620ba68849f2bb7a1427ddf",
		targetChecksum: "11f5a4052b546f70efe4699259c8276afe421599f620ba68849f2bb7a1427ddf",
		schemaCheckSQL: `SELECT
  to_regclass('public.teams') IS NOT NULL
  AND to_regclass('public.team_memberships') IS NOT NULL
  AND to_regclass('public.team_invitations') IS NOT NULL
  AND to_regclass('public.team_ownership_transfers') IS NOT NULL
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'teams'
         AND column_name IN ('id', 'name', 'status', 'member_limit',
                             'default_daily_limit_usd', 'default_weekly_limit_usd',
                             'default_monthly_limit_usd', 'created_at', 'updated_at', 'deleted_at')) = 10
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'team_memberships'
         AND column_name IN ('id', 'team_id', 'user_id', 'role',
                             'daily_limit_usd', 'weekly_limit_usd', 'monthly_limit_usd',
                             'daily_usage_usd', 'weekly_usage_usd', 'monthly_usage_usd',
                             'daily_window_start', 'weekly_window_start', 'monthly_window_start',
                             'joined_at', 'left_at', 'created_at', 'updated_at')) = 17
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'team_invitations'
         AND column_name IN ('id', 'team_id', 'inviter_user_id', 'email', 'token_hash', 'status',
                             'expires_at', 'accepted_by_user_id', 'accepted_at', 'created_at', 'updated_at')) = 11
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'team_ownership_transfers'
         AND column_name IN ('id', 'team_id', 'from_user_id', 'to_user_id', 'token_hash', 'status',
                             'expires_at', 'resolved_at', 'created_at', 'updated_at')) = 10
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public'
         AND (table_name, column_name) IN (
           ('api_keys', 'team_id'), ('api_keys', 'team_owner_disabled'),
           ('usage_logs', 'billing_user_id'), ('usage_logs', 'team_id'),
           ('batch_image_jobs', 'billing_user_id'), ('batch_image_jobs', 'team_id'),
           ('batch_image_jobs', 'allowance_reserved')
         )) = 7
  AND (SELECT COUNT(*) FROM pg_constraint c
       JOIN pg_class t ON t.oid = c.conrelid
       JOIN pg_namespace n ON n.oid = t.relnamespace
       WHERE n.nspname = 'public'
         AND c.conname IN ('api_keys_team_id_fkey', 'usage_logs_billing_user_id_fkey',
                           'usage_logs_team_id_fkey', 'batch_image_jobs_billing_user_id_fkey',
                           'batch_image_jobs_team_id_fkey')
         AND c.contype = 'f' AND c.confdeltype = 'r' AND c.convalidated) = 5
  AND (SELECT COUNT(*) FROM pg_class idx
       JOIN pg_index i ON i.indexrelid = idx.oid
       JOIN pg_namespace n ON n.oid = idx.relnamespace
       WHERE n.nspname = 'public'
         AND idx.relname IN ('team_memberships_active_user_uq', 'team_memberships_active_team_user_uq',
                             'team_memberships_active_owner_uq', 'team_memberships_team_idx',
                             'team_invitations_token_hash_uq', 'team_invitations_pending_email_uq',
                             'team_invitations_email_status_idx', 'team_ownership_transfers_token_hash_uq',
                             'team_ownership_transfers_pending_team_uq')
         AND i.indisvalid AND i.indisready) = 9
  AND EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public' AND p.proname = 'fill_usage_log_team_attribution'
      AND pg_get_functiondef(p.oid) LIKE '%NEW.billing_user_id%'
      AND pg_get_functiondef(p.oid) LIKE '%team_memberships%'
  )
  AND EXISTS (
    SELECT 1 FROM pg_trigger trg
    JOIN pg_class t ON t.oid = trg.tgrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = 'public' AND t.relname = 'usage_logs'
      AND trg.tgname = 'usage_logs_fill_team_attribution'
      AND NOT trg.tgisinternal AND trg.tgenabled <> 'D'
  )`,
	},
	{
		legacyFilename: "192_harden_team_lifecycle.sql",
		targetFilename: "222_harden_team_lifecycle.sql",
		legacyChecksum: "686cb8a5794ca2534192fcb35efca28f70ffd9896e1e0dacbf5aa4c9376100e6",
		targetChecksum: "686cb8a5794ca2534192fcb35efca28f70ffd9896e1e0dacbf5aa4c9376100e6",
		schemaCheckSQL: `SELECT
  EXISTS (
    SELECT 1 FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public' AND p.proname = 'protect_team_membership_on_user_delete'
      AND pg_get_functiondef(p.oid) LIKE '%TEAM_OWNER_TRANSFER_REQUIRED%'
      AND pg_get_functiondef(p.oid) LIKE '%UPDATE team_memberships%'
      AND pg_get_functiondef(p.oid) LIKE '%UPDATE api_keys%'
  )
  AND (SELECT COUNT(*) FROM pg_trigger trg
       JOIN pg_class t ON t.oid = trg.tgrelid
       JOIN pg_namespace n ON n.oid = t.relnamespace
       WHERE n.nspname = 'public' AND t.relname = 'users'
         AND trg.tgname IN ('users_protect_team_membership_soft_delete',
                            'users_protect_team_membership_hard_delete')
         AND NOT trg.tgisinternal AND trg.tgenabled <> 'D') = 2`,
	},
	{
		legacyFilename: "193_add_team_attribution_indexes_notx.sql",
		targetFilename: "223_add_team_attribution_indexes_notx.sql",
		legacyChecksum: "52ed77f25554b32d55b9bd1bec90cee503481baf0010cd64154f8d56dba212ca",
		targetChecksum: "52ed77f25554b32d55b9bd1bec90cee503481baf0010cd64154f8d56dba212ca",
		schemaCheckSQL: `SELECT COUNT(*) = 5
FROM pg_class idx
JOIN pg_index i ON i.indexrelid = idx.oid
JOIN pg_namespace n ON n.oid = idx.relnamespace
WHERE n.nspname = 'public'
  AND idx.relname IN ('idx_api_keys_team_id_active', 'idx_usage_logs_billing_user_created',
                      'idx_usage_logs_team_created', 'idx_batch_image_jobs_billing_user_created',
                      'idx_batch_image_jobs_team_created')
  AND i.indisvalid AND i.indisready`,
	},
	{
		legacyFilename: "196_add_image_objects.sql",
		targetFilename: "224_add_image_objects.sql",
		legacyChecksum: "66b8d271b451cf261da8abf1525feaeaa1058d4ddde061de6ff9cf1b6404a64a",
		targetChecksum: "66b8d271b451cf261da8abf1525feaeaa1058d4ddde061de6ff9cf1b6404a64a",
		schemaCheckSQL: `SELECT
  to_regclass('public.image_objects') IS NOT NULL
  AND (SELECT COUNT(*) FROM information_schema.columns
       WHERE table_schema = 'public' AND table_name = 'image_objects'
         AND column_name IN ('id', 'object_id', 'user_id', 'api_key_id', 'task_id',
                             'storage_key', 'content_type', 'byte_size', 'created_at')) = 9
  AND (SELECT COUNT(*) FROM pg_class idx
       JOIN pg_index i ON i.indexrelid = idx.oid
       JOIN pg_namespace n ON n.oid = idx.relnamespace
       WHERE n.nspname = 'public'
         AND idx.relname IN ('image_objects_pkey', 'image_objects_object_id_key',
                             'image_objects_storage_key_key', 'image_objects_user_id_created_at_idx',
                             'image_objects_task_id_idx', 'image_objects_api_key_id_created_at_idx')
         AND i.indisvalid AND i.indisready) = 6
  AND EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = 'public' AND t.relname = 'image_objects'
      AND c.contype = 'c' AND c.convalidated
      AND pg_get_constraintdef(c.oid) LIKE '%byte_size >= 0%'
  )`,
	},
}

func prepareLegacyMigrationLineage(ctx context.Context, db migrationConnection, fsys fs.FS) error {
	if len(legacyMigrationLineageRules) == 0 {
		return nil
	}

	_, err := fs.Stat(fsys, legacyMigrationLineageRules[0].targetFilename)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect legacy migration lineage activation: %w", err)
	}

	return prepareLegacyMigrationLineageRules(ctx, db, fsys, legacyMigrationLineageRules)
}

func prepareLegacyMigrationLineageRules(
	ctx context.Context,
	db migrationConnection,
	fsys fs.FS,
	rules []legacyMigrationLineageRule,
) error {
	type equivalentMigration struct {
		filename string
		checksum string
	}

	pending := make([]equivalentMigration, 0, len(rules))
	for _, rule := range rules {
		content, err := fs.ReadFile(fsys, rule.targetFilename)
		if err != nil {
			return fmt.Errorf("read lineage target %s: %w", rule.targetFilename, err)
		}
		actualTargetChecksum := checksumMigrationContent(string(content))
		if actualTargetChecksum != rule.targetChecksum {
			return fmt.Errorf(
				"legacy migration lineage target checksum mismatch for %s (expected=%s actual=%s)",
				rule.targetFilename,
				rule.targetChecksum,
				actualTargetChecksum,
			)
		}

		legacyChecksum, legacyExists, err := migrationRecordChecksum(ctx, db, rule.legacyFilename)
		if err != nil {
			return fmt.Errorf("check legacy migration %s: %w", rule.legacyFilename, err)
		}
		targetChecksum, targetExists, err := migrationRecordChecksum(ctx, db, rule.targetFilename)
		if err != nil {
			return fmt.Errorf("check lineage target migration %s: %w", rule.targetFilename, err)
		}

		if targetExists && targetChecksum != rule.targetChecksum {
			return fmt.Errorf(
				"legacy migration lineage database checksum mismatch for %s (expected=%s actual=%s)",
				rule.targetFilename,
				rule.targetChecksum,
				targetChecksum,
			)
		}
		if !legacyExists {
			continue
		}
		if legacyChecksum != rule.legacyChecksum {
			return fmt.Errorf(
				"legacy migration checksum mismatch for %s (expected=%s actual=%s)",
				rule.legacyFilename,
				rule.legacyChecksum,
				legacyChecksum,
			)
		}

		var schemaMatches bool
		if err := db.QueryRowContext(ctx, rule.schemaCheckSQL).Scan(&schemaMatches); err != nil {
			return fmt.Errorf("verify legacy migration schema for %s: %w", rule.legacyFilename, err)
		}
		if !schemaMatches {
			return fmt.Errorf(
				"legacy migration schema mismatch for %s; refusing to record equivalent %s",
				rule.legacyFilename,
				rule.targetFilename,
			)
		}
		if !targetExists {
			pending = append(pending, equivalentMigration{
				filename: rule.targetFilename,
				checksum: rule.targetChecksum,
			})
		}
	}

	if len(pending) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy migration lineage registration: %w", err)
	}
	for _, migration := range pending {
		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)",
			migration.filename,
			migration.checksum,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record equivalent migration %s: %w", migration.filename, err)
		}
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("commit legacy migration lineage registration: %w", err)
	}

	return nil
}

func migrationRecordChecksum(
	ctx context.Context,
	db migrationConnection,
	filename string,
) (string, bool, error) {
	var checksum string
	err := db.QueryRowContext(
		ctx,
		"SELECT checksum FROM schema_migrations WHERE filename = $1",
		filename,
	).Scan(&checksum)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return checksum, true, nil
}

func checksumMigrationContent(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return fmt.Sprintf("%x", sum)
}
