-- Owner 必须先转让或解散；普通成员删除时自动离队并禁用其团队 Key。
CREATE OR REPLACE FUNCTION protect_team_membership_on_user_delete()
RETURNS TRIGGER AS $$
DECLARE
    removed_at TIMESTAMPTZ;
BEGIN
    IF EXISTS (
        SELECT 1
        FROM team_memberships tm
        JOIN teams t ON t.id = tm.team_id
        WHERE tm.user_id = OLD.id
          AND tm.left_at IS NULL
          AND tm.role = 'owner'
          AND t.deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION 'TEAM_OWNER_TRANSFER_REQUIRED' USING ERRCODE = '23514';
    END IF;

    removed_at := CASE WHEN TG_OP = 'DELETE' THEN NOW() ELSE COALESCE(NEW.deleted_at, NOW()) END;
    UPDATE team_memberships
    SET left_at = removed_at, updated_at = removed_at
    WHERE user_id = OLD.id AND left_at IS NULL AND role = 'member';

    UPDATE api_keys
    SET status = 'disabled', updated_at = removed_at
    WHERE user_id = OLD.id AND team_id IS NOT NULL AND deleted_at IS NULL;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_protect_team_membership_soft_delete ON users;
CREATE TRIGGER users_protect_team_membership_soft_delete
    BEFORE UPDATE OF deleted_at ON users
    FOR EACH ROW
    WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
    EXECUTE FUNCTION protect_team_membership_on_user_delete();

DROP TRIGGER IF EXISTS users_protect_team_membership_hard_delete ON users;
CREATE TRIGGER users_protect_team_membership_hard_delete
    BEFORE DELETE ON users
    FOR EACH ROW EXECUTE FUNCTION protect_team_membership_on_user_delete();
