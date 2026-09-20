-- Plus 218 is immutable and still clears video prices on every non-Grok,
-- non-composite group. GotoCC Jimeng video billing lives on OpenAI-platform
-- groups (LC-002) and must not follow that cleanup.
--
-- 218 writes groups_video_price_backup_218 before the UPDATE. This compensating
-- migration restores only OpenAI-platform rows from that snapshot. Grok and
-- composite groups were never cleared. Other non-OpenAI leftovers stay cleared.

UPDATE groups g
SET video_price_480p = b.video_price_480p,
    video_price_720p = b.video_price_720p,
    video_price_1080p = b.video_price_1080p,
    video_model_prices = b.video_model_prices
FROM groups_video_price_backup_218 b
WHERE g.id = b.group_id
  AND b.platform = 'openai'
  AND g.platform = 'openai';
