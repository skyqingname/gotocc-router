ALTER TABLE groups ADD COLUMN rate_schedule JSONB NOT NULL DEFAULT '{"enabled":false,"timezone":"","rules":[]}'::jsonb;

-- Preserve the old subscription-only daily window; the empty zone explicitly
-- inherits the configured server timezone, as the old implementation did.
UPDATE groups SET rate_schedule = jsonb_build_object(
  'enabled', peak_rate_enabled,
  'timezone', '',
  'rules', jsonb_build_array(jsonb_build_object(
    'id', 'legacy-peak', 'enabled', true, 'start', peak_start,
    'end', peak_end, 'multiplier', peak_rate_multiplier
  ))
) WHERE subscription_type = 'subscription' AND peak_start <> '' AND peak_end <> '';
