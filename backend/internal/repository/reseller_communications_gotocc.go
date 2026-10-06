package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

const resellerAnnouncementColumns = `a.id,a.title,a.content,a.status,a.notify_mode,a.targeting,
    a.starts_at,a.ends_at,a.created_at,a.updated_at,a.owner_user_id,a.owner_user_id,a.source_announcement_id`

const resellerMainAnnouncementColumns = `a.id,a.title,a.content,a.status,a.notify_mode,a.targeting,
    a.starts_at,a.ends_at,a.created_at,a.updated_at,a.created_by,a.updated_by,NULL::bigint`

const resellerVisibleAnnouncementFrom = ` FROM reseller_announcements a
    JOIN reseller_customers c ON c.owner_user_id=a.owner_user_id AND c.user_id=$1
    JOIN reseller_profiles p ON p.user_id=a.owner_user_id
    LEFT JOIN announcements source ON source.id=a.source_announcement_id
    WHERE p.enabled AND p.announcements_enabled AND a.status='active'
    AND (a.starts_at IS NULL OR a.starts_at<=NOW()) AND (a.ends_at IS NULL OR a.ends_at>NOW())
    AND (a.source_announcement_id IS NULL OR (source.status='active'
        AND EXISTS(SELECT 1 FROM reseller_announcement_reviews review WHERE review.owner_user_id=a.owner_user_id
            AND review.source_announcement_id=a.source_announcement_id AND review.status='approved')
        AND (source.starts_at IS NULL OR source.starts_at<=NOW())
        AND (source.ends_at IS NULL OR source.ends_at>NOW())))`

func scanResellerAnnouncement(row interface{ Scan(...any) error }, additional ...any) (*service.ResellerAnnouncement, error) {
	a := &service.ResellerAnnouncement{}
	var targeting []byte
	dest := []any{&a.ID, &a.Title, &a.Content, &a.Status, &a.NotifyMode, &targeting,
		&a.StartsAt, &a.EndsAt, &a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy, &a.SourceAnnouncementID}
	if err := row.Scan(append(dest, additional...)...); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(targeting, &a.Targeting); err != nil {
		return nil, err
	}
	return a, nil
}

func (r *resellerRepository) SaveCommunicationSettings(ctx context.Context, ownerID int64, settings service.ResellerCommunicationSettings) error {
	_, err := r.executor(ctx).ExecContext(ctx, `UPDATE reseller_profiles SET
        contact_enabled=$2,contact_info=$3,announcements_enabled=$4,sync_main_announcements=$5,updated_at=NOW()
        WHERE user_id=$1`, ownerID, settings.ContactEnabled, settings.ContactInfo, settings.AnnouncementsEnabled, settings.SyncMainAnnouncements)
	return err
}

func (r *resellerRepository) ListResellerAnnouncements(ctx context.Context, ownerID int64) ([]service.ResellerAnnouncement, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT `+resellerAnnouncementColumns+`
        FROM reseller_announcements a WHERE a.owner_user_id=$1 ORDER BY a.updated_at DESC,a.id DESC LIMIT $2`, ownerID, reseller.AnnouncementListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []service.ResellerAnnouncement{}
	for rows.Next() {
		item, err := scanResellerAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *resellerRepository) resellerAnnouncementByID(ctx context.Context, ownerID, id int64) (*service.ResellerAnnouncement, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT `+resellerAnnouncementColumns+`
        FROM reseller_announcements a WHERE a.owner_user_id=$1 AND a.id=$2`, ownerID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAnnouncementNotFound
	}
	return scanResellerAnnouncement(rows)
}

func (r *resellerRepository) CreateResellerAnnouncement(ctx context.Context, ownerID int64, input service.ResellerAnnouncementInput) (*service.ResellerAnnouncement, error) {
	var id int64
	err := scanSingleRow(ctx, r.executor(ctx), `INSERT INTO reseller_announcements(owner_user_id,title,content,status,notify_mode)
        VALUES($1,$2,$3,$4,$5) RETURNING id`, []any{ownerID, input.Title, input.Content, input.Status, input.NotifyMode}, &id)
	if err != nil {
		return nil, err
	}
	return r.resellerAnnouncementByID(ctx, ownerID, id)
}

func (r *resellerRepository) UpdateResellerAnnouncement(ctx context.Context, ownerID, id int64, input service.ResellerAnnouncementInput) (*service.ResellerAnnouncement, error) {
	var updatedID int64
	err := scanSingleRow(ctx, r.executor(ctx), `UPDATE reseller_announcements SET title=$3,content=$4,status=$5,notify_mode=$6,updated_at=NOW()
        WHERE owner_user_id=$1 AND id=$2 AND source_announcement_id IS NULL RETURNING id`,
		[]any{ownerID, id, input.Title, input.Content, input.Status, input.NotifyMode}, &updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrAnnouncementNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.resellerAnnouncementByID(ctx, ownerID, updatedID)
}

func (r *resellerRepository) SetResellerAnnouncementStatus(ctx context.Context, ownerID, id int64, status string) error {
	var updatedID int64
	err := scanSingleRow(ctx, r.executor(ctx), `UPDATE reseller_announcements a SET status=$3,updated_at=NOW()
        WHERE owner_user_id=$1 AND id=$2 AND ($3<>'active' OR source_announcement_id IS NULL OR
            EXISTS(SELECT 1 FROM reseller_announcement_reviews review WHERE review.owner_user_id=$1
                AND review.source_announcement_id=a.source_announcement_id AND review.status='approved'))
        RETURNING id`, []any{ownerID, id, status}, &updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrAnnouncementNotFound
	}
	return err
}

func (r *resellerRepository) ListMainAnnouncementReviews(ctx context.Context, ownerID int64) ([]service.ResellerAnnouncementReview, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT `+resellerMainAnnouncementColumns+`,
        CASE WHEN review.source_updated_at=a.updated_at THEN review.status ELSE 'pending' END,review.reviewed_at
        FROM announcements a LEFT JOIN reseller_announcement_reviews review
            ON review.source_announcement_id=a.id AND review.owner_user_id=$1
        WHERE a.status='active' AND (a.ends_at IS NULL OR a.ends_at>NOW())
        ORDER BY a.updated_at DESC,a.id DESC LIMIT $2`, ownerID, reseller.AnnouncementListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []service.ResellerAnnouncementReview{}
	for rows.Next() {
		item := service.ResellerAnnouncementReview{}
		announcement, err := scanResellerAnnouncement(rows, &item.ReviewStatus, &item.ReviewedAt)
		if err != nil {
			return nil, err
		}
		item.Announcement = announcement.Announcement
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *resellerRepository) ReviewMainAnnouncement(ctx context.Context, ownerID, sourceID int64, version time.Time, status string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var syncEnabled bool
	err = tx.QueryRowContext(ctx, `SELECT sync_main_announcements FROM reseller_profiles
        WHERE user_id=$1 AND enabled FOR SHARE`, ownerID).Scan(&syncEnabled)
	if err != nil {
		return err
	}
	if !syncEnabled {
		return infraerrors.BadRequest("RESELLER_ANNOUNCEMENT_SYNC_DISABLED", "主站公告同步已关闭")
	}
	source, err := scanResellerAnnouncement(tx.QueryRowContext(ctx, `SELECT `+resellerMainAnnouncementColumns+`
        FROM announcements a WHERE a.id=$1 AND a.status='active' AND (a.ends_at IS NULL OR a.ends_at>NOW()) FOR SHARE`, sourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrAnnouncementNotFound
	}
	if err != nil {
		return err
	}
	if !source.UpdatedAt.Equal(version) {
		return infraerrors.Conflict("ANNOUNCEMENT_REVIEW_VERSION_CHANGED", "主站公告已更新，请重新查看后审核")
	}
	if status == "approved" {
		targeting, err := json.Marshal(source.Targeting)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reseller_announcements
            (owner_user_id,source_announcement_id,title,content,status,notify_mode,targeting,starts_at,ends_at)
            VALUES($1,$2,$3,$4,'active',$5,$6,$7,$8)
            ON CONFLICT(owner_user_id,source_announcement_id) DO UPDATE SET
                title=EXCLUDED.title,content=EXCLUDED.content,status='active',notify_mode=EXCLUDED.notify_mode,
                targeting=EXCLUDED.targeting,starts_at=EXCLUDED.starts_at,ends_at=EXCLUDED.ends_at,updated_at=NOW()`,
			ownerID, sourceID, source.Title, source.Content, source.NotifyMode, string(targeting), source.StartsAt, source.EndsAt)
		if err != nil {
			return err
		}
	} else {
		if _, err = tx.ExecContext(ctx, `UPDATE reseller_announcements SET status='archived',updated_at=NOW()
            WHERE owner_user_id=$1 AND source_announcement_id=$2`, ownerID, sourceID); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reseller_announcement_reviews(owner_user_id,source_announcement_id,source_updated_at,status)
        VALUES($1,$2,$3,$4) ON CONFLICT(owner_user_id,source_announcement_id) DO UPDATE SET
        source_updated_at=EXCLUDED.source_updated_at,status=EXCLUDED.status,reviewed_at=NOW()`, ownerID, sourceID, version, status)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *resellerRepository) ListCustomerAnnouncements(ctx context.Context, userID int64) ([]service.UserAnnouncement, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT `+resellerAnnouncementColumns+`,
        (SELECT read_at FROM reseller_announcement_reads ar WHERE ar.announcement_id=a.id AND ar.user_id=$1 AND ar.read_at>=a.updated_at)`+
		resellerVisibleAnnouncementFrom+` ORDER BY a.updated_at DESC,a.id DESC LIMIT $2`, userID, reseller.AnnouncementListLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []service.UserAnnouncement{}
	for rows.Next() {
		var readAt *time.Time
		announcement, err := scanResellerAnnouncement(rows, &readAt)
		if err != nil {
			return nil, err
		}
		items = append(items, service.UserAnnouncement{Announcement: announcement.Announcement, ReadAt: readAt})
	}
	return items, rows.Err()
}

func (r *resellerRepository) MarkCustomerAnnouncementRead(ctx context.Context, userID, announcementID int64) error {
	result, err := r.executor(ctx).ExecContext(ctx, `INSERT INTO reseller_announcement_reads(announcement_id,user_id,read_at)
        SELECT a.id,$1,NOW()`+resellerVisibleAnnouncementFrom+` AND a.id=$2
        ON CONFLICT(announcement_id,user_id) DO UPDATE SET read_at=EXCLUDED.read_at`, userID, announcementID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrAnnouncementNotFound
	}
	return nil
}
