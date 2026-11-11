-- Independent, shared request capacity for each team. Historical personal
-- settings and balances are untouched. Zero retains the existing unlimited meaning.
ALTER TABLE teams
    ADD COLUMN concurrency INTEGER NOT NULL DEFAULT 0 CHECK (concurrency >= 0),
    ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0 CHECK (rpm_limit >= 0);

-- Snapshot the current owner once. Dissolved teams with no current owner stay
-- inactive. Legacy nonpositive limits mean unlimited and normalize to zero.
UPDATE teams t
SET concurrency = GREATEST(u.concurrency, 0), rpm_limit = GREATEST(u.rpm_limit, 0)
FROM team_memberships m JOIN users u ON u.id = m.user_id
WHERE m.team_id = t.id AND m.role = 'owner' AND m.left_at IS NULL;
