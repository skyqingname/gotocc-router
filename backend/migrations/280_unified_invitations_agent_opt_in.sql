-- LC-018/020: one invitation directory for admission and referral ownership.
-- Retain existing code values, limits, ownership and historical attribution.
INSERT INTO reusable_invitation_codes (code, owner_user_id, status, max_uses, used_count, notes, created_at, updated_at)
SELECT aff_code, user_id, 'active', 0, 0, '', created_at, NOW()
FROM user_affiliates
ON CONFLICT (code) DO NOTHING;

-- LC-024: these users already submitted their intent. Activation starts now;
-- no historical commission entries are created or changed.
UPDATE agent_profiles
SET status = 'approved', reviewed_at = NOW(), reviewed_by = NULL, updated_at = NOW()
WHERE status IN ('pending', 'rejected');

COMMENT ON TABLE agent_profiles IS 'LC-024 代理主动开通；用户点击申请立即生效，无人工审核与退出';
COMMENT ON COLUMN agent_profiles.reviewed_at IS '代理资格生效时间；保留历史列名';
