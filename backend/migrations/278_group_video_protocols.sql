ALTER TABLE groups ADD COLUMN video_models JSONB NOT NULL DEFAULT '{}'::jsonb;
-- Move existing Video model contracts to each bound group; keep the original
-- channel JSON for historical compatibility, without reading it at runtime.
UPDATE groups g SET video_models=c.features_config->'video_models'
FROM channel_groups cg JOIN channels c ON c.id=cg.channel_id
WHERE g.id=cg.group_id AND g.platform='video'
AND jsonb_typeof(c.features_config->'video_models')='object';
