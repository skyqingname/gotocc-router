-- LC-020: current attribution is separate from the immutable registration source.
ALTER TABLE user_affiliates
 ADD COLUMN attribution_code_type TEXT NOT NULL DEFAULT '',
 ADD COLUMN attribution_code TEXT NOT NULL DEFAULT '',
 ADD COLUMN inviter_version BIGINT NOT NULL DEFAULT 0,
 ADD COLUMN inviter_effective_at TIMESTAMPTZ;

UPDATE user_affiliates ua
SET attribution_code_type = CASE WHEN rc.id IS NOT NULL THEN 'permanent' ELSE 'aff' END,
    attribution_code = CASE WHEN rc.id IS NOT NULL THEN rc.code ELSE COALESCE(inviter.aff_code, '') END
FROM user_affiliates inviter
LEFT JOIN reusable_invitation_codes rc ON FALSE
WHERE FALSE;

UPDATE user_affiliates ua
SET attribution_code_type = CASE WHEN EXISTS (
      SELECT 1 FROM reusable_invitation_codes rc
      WHERE UPPER(rc.code) = UPPER(ua.invitation_code) AND rc.owner_user_id = ua.inviter_id
    ) THEN 'permanent' ELSE 'aff' END,
    attribution_code = CASE WHEN EXISTS (
      SELECT 1 FROM reusable_invitation_codes rc
      WHERE UPPER(rc.code) = UPPER(ua.invitation_code) AND rc.owner_user_id = ua.inviter_id
    ) THEN ua.invitation_code ELSE inviter.aff_code END
FROM user_affiliates inviter
WHERE ua.inviter_id = inviter.user_id;

CREATE TABLE affiliate_inviter_changes (
 id BIGSERIAL PRIMARY KEY,
 user_id BIGINT NOT NULL,
 version BIGINT NOT NULL,
 previous_inviter JSONB,
 inviter JSONB NOT NULL,
 previous_code_type TEXT NOT NULL,
 previous_code TEXT NOT NULL,
 code_type TEXT NOT NULL,
 code TEXT NOT NULL,
 actor_user_id BIGINT,
 auth_method TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(user_id, version)
);

CREATE TABLE affiliate_payment_attributions (
 order_id BIGINT PRIMARY KEY REFERENCES payment_orders(id) ON DELETE CASCADE,
 user_id BIGINT NOT NULL,
 inviter_ids JSONB NOT NULL,
 capture_source TEXT NOT NULL,
 captured_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

-- Freeze the pre-feature graph for already-credited unfinished legacy orders.
-- This is an upgrade-time baseline, not a reconstruction of historical credit time.
-- Generation count below is fixed from affiliate-defaults.json when authored.
WITH RECURSIVE pending AS (
 SELECT o.id, o.user_id
 FROM payment_orders o
 JOIN redeem_codes rc ON rc.code = o.recharge_code AND rc.status = 'used' AND rc.used_by = o.user_id
 WHERE o.order_type = 'balance' AND o.amount > 0 AND o.pay_amount > 0
   AND o.paid_at IS NOT NULL AND o.status <> 'COMPLETED'
   AND NOT EXISTS (
     SELECT 1 FROM payment_audit_logs a WHERE a.order_id = o.id::text
       AND a.action IN ('AFFILIATE_REBATE_APPLIED', 'AFFILIATE_REBATE_SKIPPED')
   )
), chain AS (
 SELECT p.id, ua.inviter_id, 1 AS level, ARRAY[p.user_id, ua.inviter_id] AS path
 FROM pending p JOIN user_affiliates ua ON ua.user_id = p.user_id
 WHERE ua.inviter_id IS NOT NULL AND ua.inviter_id <> p.user_id
 UNION ALL
 SELECT c.id, ua.inviter_id, c.level + 1, c.path || ua.inviter_id
 FROM chain c JOIN user_affiliates ua ON ua.user_id = c.inviter_id
 WHERE ua.inviter_id IS NOT NULL AND NOT ua.inviter_id = ANY(c.path)
   AND c.level < 3
)
INSERT INTO affiliate_payment_attributions(order_id, user_id, inviter_ids, capture_source)
SELECT p.id, p.user_id,
 COALESCE((SELECT jsonb_agg(c.inviter_id ORDER BY c.level) FROM chain c WHERE c.id = p.id), '[]'::jsonb),
 'legacy_upgrade'
FROM pending p;
