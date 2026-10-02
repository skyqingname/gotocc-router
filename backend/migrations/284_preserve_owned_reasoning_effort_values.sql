-- GoToCC 283 introduced generic maps before upstream 271. Preserve the
-- previous explicit max value when the new map has no max entry, matching
-- the owned runtime precedence. Existing generic max entries win.
UPDATE channel_model_pricing
SET reasoning_effort_multipliers = reasoning_effort_multipliers ||
    jsonb_build_object('max', max_reasoning_effort_multiplier)
WHERE max_reasoning_effort_multiplier IS NOT NULL
  AND NOT (reasoning_effort_multipliers ? 'max');

UPDATE groups AS g
SET model_pricing = (
    SELECT jsonb_agg(
        CASE
            WHEN jsonb_typeof(entry) = 'object'
                 AND jsonb_typeof(entry->'max_reasoning_effort_multiplier') = 'number'
                 AND NOT (COALESCE(entry->'reasoning_effort_multipliers', '{}'::jsonb) ? 'max') THEN
                entry || jsonb_build_object('reasoning_effort_multipliers',
                    COALESCE(NULLIF(entry->'reasoning_effort_multipliers', 'null'::jsonb), '{}'::jsonb) ||
                    jsonb_build_object('max', entry->'max_reasoning_effort_multiplier'))
            ELSE entry
        END ORDER BY ordinal
    )
    FROM jsonb_array_elements(g.model_pricing) WITH ORDINALITY AS pricing(entry, ordinal)
)
WHERE jsonb_typeof(g.model_pricing) = 'array'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(
          CASE WHEN jsonb_typeof(g.model_pricing) = 'array' THEN g.model_pricing ELSE '[]'::jsonb END
      ) AS pricing(entry)
      WHERE jsonb_typeof(entry->'max_reasoning_effort_multiplier') = 'number'
  );
