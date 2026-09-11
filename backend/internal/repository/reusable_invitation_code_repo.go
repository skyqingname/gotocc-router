package repository

import (
	"context"
	"strings"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/predicate"
	"github.com/LuckyKuang/sub2api-plus/ent/reusableinvitationcode"
	"github.com/LuckyKuang/sub2api-plus/ent/reusableinvitationcodeuse"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	entsql "entgo.io/ent/dialect/sql"
)

const defaultReusableInvitationCodeUseLimit = 50

type reusableInvitationCodeRepository struct {
	client *dbent.Client
}

func NewReusableInvitationCodeRepository(client *dbent.Client) service.ReusableInvitationCodeRepository {
	return &reusableInvitationCodeRepository{client: client}
}

func (r *reusableInvitationCodeRepository) Create(ctx context.Context, code *service.ReusableInvitationCode) error {
	created, err := clientFromContext(ctx, r.client).ReusableInvitationCode.Create().
		SetCode(code.Code).
		SetStatus(code.Status).
		SetMaxUses(code.MaxUses).
		SetUsedCount(code.UsedCount).
		SetNillableExpiresAt(code.ExpiresAt).
		SetNotes(code.Notes).
		Save(ctx)
	if err != nil {
		return err
	}
	applyReusableInvitationCodeEntity(code, created)
	return nil
}

func (r *reusableInvitationCodeRepository) GetByID(ctx context.Context, id int64) (*service.ReusableInvitationCode, error) {
	m, err := clientFromContext(ctx, r.client).ReusableInvitationCode.Query().
		Where(reusableinvitationcode.IDEQ(id)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrReusableInvitationCodeNotFound
		}
		return nil, err
	}
	return reusableInvitationCodeEntityToService(m), nil
}

func (r *reusableInvitationCodeRepository) GetByCode(ctx context.Context, code string) (*service.ReusableInvitationCode, error) {
	m, err := clientFromContext(ctx, r.client).ReusableInvitationCode.Query().
		Where(reusableinvitationcode.CodeEQ(code)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrReusableInvitationCodeNotFound
		}
		return nil, err
	}
	return reusableInvitationCodeEntityToService(m), nil
}

func (r *reusableInvitationCodeRepository) GetUsableByCode(ctx context.Context, code string) (*service.ReusableInvitationCode, error) {
	got, err := r.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !got.IsUsableAt(time.Now()) {
		return nil, service.ErrReusableInvitationCodeInvalid
	}
	return got, nil
}

func (r *reusableInvitationCodeRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.ReusableInvitationCode, *pagination.PaginationResult, error) {
	q := clientFromContext(ctx, r.client).ReusableInvitationCode.Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	codes, err := q.Offset(params.Offset()).Limit(params.Limit()).
		Order(reusableInvitationCodeListOrder(params)...).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return reusableInvitationCodeEntitiesToService(codes), paginationResultFromTotal(int64(total), params), nil
}

func (r *reusableInvitationCodeRepository) Disable(ctx context.Context, id int64) (*service.ReusableInvitationCode, error) {
	updated, err := clientFromContext(ctx, r.client).ReusableInvitationCode.UpdateOneID(id).
		SetStatus(service.ReusableInvitationCodeStatusDisabled).Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrReusableInvitationCodeNotFound
		}
		return nil, err
	}
	return reusableInvitationCodeEntityToService(updated), nil
}

func (r *reusableInvitationCodeRepository) Use(ctx context.Context, id, userID int64, email, authSource string) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return r.use(ctx, tx.Client(), id, userID, email, authSource)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	defer func() { _ = tx.Rollback() }()
	if err := r.use(txCtx, tx.Client(), id, userID, email, authSource); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *reusableInvitationCodeRepository) Release(ctx context.Context, id, userID int64) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return r.release(ctx, tx.Client(), id, userID)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	defer func() { _ = tx.Rollback() }()
	if err := r.release(txCtx, tx.Client(), id, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *reusableInvitationCodeRepository) release(ctx context.Context, client *dbent.Client, id, userID int64) error {
	use, err := client.ReusableInvitationCodeUse.Query().Where(
		reusableinvitationcodeuse.CodeIDEQ(id),
		reusableinvitationcodeuse.UserIDEQ(userID),
	).Order(dbent.Desc(reusableinvitationcodeuse.FieldID)).First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := client.ReusableInvitationCodeUse.DeleteOneID(use.ID).Exec(ctx); err != nil {
		return err
	}
	affected, err := client.ReusableInvitationCode.Update().Where(
		reusableinvitationcode.IDEQ(id),
		reusableinvitationcode.UsedCountGT(0),
	).AddUsedCount(-1).Save(ctx)
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrReusableInvitationCodeInvalid
	}
	return nil
}

func (r *reusableInvitationCodeRepository) use(ctx context.Context, client *dbent.Client, id, userID int64, email, authSource string) error {
	now := time.Now()
	affected, err := client.ReusableInvitationCode.Update().Where(
		reusableinvitationcode.IDEQ(id),
		reusableinvitationcode.StatusEQ(service.ReusableInvitationCodeStatusActive),
		reusableinvitationcode.Or(reusableinvitationcode.ExpiresAtIsNil(), reusableinvitationcode.ExpiresAtGT(now)),
		reusableinvitationcode.Or(reusableinvitationcode.MaxUsesEQ(0), usedCountLTMaxUses()),
	).AddUsedCount(1).Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return reusableInvitationCodeUseConflictError(ctx, client, id, now)
	}
	_, err = client.ReusableInvitationCodeUse.Create().
		SetCodeID(id).SetUserID(userID).SetEmail(email).SetAuthSource(authSource).Save(ctx)
	return err
}

func reusableInvitationCodeUseConflictError(ctx context.Context, client *dbent.Client, id int64, now time.Time) error {
	code, err := client.ReusableInvitationCode.Query().Where(reusableinvitationcode.IDEQ(id)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrReusableInvitationCodeNotFound
		}
		return err
	}
	if code.Status != service.ReusableInvitationCodeStatusActive || (code.ExpiresAt != nil && !code.ExpiresAt.After(now)) {
		return service.ErrReusableInvitationCodeInvalid
	}
	if code.MaxUses > 0 && code.UsedCount >= code.MaxUses {
		return service.ErrReusableInvitationCodeExhausted
	}
	return service.ErrReusableInvitationCodeInvalid
}

func (r *reusableInvitationCodeRepository) ListUsesByCodeID(ctx context.Context, codeID int64, limit int) ([]service.ReusableInvitationCodeUse, error) {
	if limit <= 0 {
		limit = defaultReusableInvitationCodeUseLimit
	}
	if limit > 1000 {
		limit = 1000
	}
	uses, err := clientFromContext(ctx, r.client).ReusableInvitationCodeUse.Query().
		Where(reusableinvitationcodeuse.CodeIDEQ(codeID)).
		Order(dbent.Desc(reusableinvitationcodeuse.FieldUsedAt), dbent.Desc(reusableinvitationcodeuse.FieldID)).
		Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	return reusableInvitationCodeUseEntitiesToService(uses), nil
}

func reusableInvitationCodeListOrder(params pagination.PaginationParams) []reusableinvitationcode.OrderOption {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := reusableinvitationcode.FieldID
	switch sortBy {
	case "code":
		field = reusableinvitationcode.FieldCode
	case "status":
		field = reusableinvitationcode.FieldStatus
	case "max_uses":
		field = reusableinvitationcode.FieldMaxUses
	case "used_count":
		field = reusableinvitationcode.FieldUsedCount
	case "expires_at":
		field = reusableinvitationcode.FieldExpiresAt
	case "created_at":
		field = reusableinvitationcode.FieldCreatedAt
	case "updated_at":
		field = reusableinvitationcode.FieldUpdatedAt
	}
	if sortOrder == pagination.SortOrderAsc {
		return []reusableinvitationcode.OrderOption{dbent.Asc(field), dbent.Asc(reusableinvitationcode.FieldID)}
	}
	return []reusableinvitationcode.OrderOption{dbent.Desc(field), dbent.Desc(reusableinvitationcode.FieldID)}
}

func usedCountLTMaxUses() predicate.ReusableInvitationCode {
	return predicate.ReusableInvitationCode(entsql.FieldsLT(reusableinvitationcode.FieldUsedCount, reusableinvitationcode.FieldMaxUses))
}

func reusableInvitationCodeEntityToService(m *dbent.ReusableInvitationCode) *service.ReusableInvitationCode {
	if m == nil {
		return nil
	}
	code := &service.ReusableInvitationCode{}
	applyReusableInvitationCodeEntity(code, m)
	return code
}

func applyReusableInvitationCodeEntity(dst *service.ReusableInvitationCode, m *dbent.ReusableInvitationCode) {
	dst.ID, dst.Code, dst.Status = m.ID, m.Code, m.Status
	dst.MaxUses, dst.UsedCount = m.MaxUses, m.UsedCount
	dst.ExpiresAt, dst.Notes = m.ExpiresAt, m.Notes
	dst.CreatedAt, dst.UpdatedAt = m.CreatedAt, m.UpdatedAt
}

func reusableInvitationCodeEntitiesToService(items []*dbent.ReusableInvitationCode) []service.ReusableInvitationCode {
	out := make([]service.ReusableInvitationCode, 0, len(items))
	for _, item := range items {
		if code := reusableInvitationCodeEntityToService(item); code != nil {
			out = append(out, *code)
		}
	}
	return out
}

func reusableInvitationCodeUseEntitiesToService(items []*dbent.ReusableInvitationCodeUse) []service.ReusableInvitationCodeUse {
	out := make([]service.ReusableInvitationCodeUse, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, service.ReusableInvitationCodeUse{
				ID: item.ID, CodeID: item.CodeID, UserID: item.UserID,
				Email: item.Email, AuthSource: item.AuthSource, UsedAt: item.UsedAt,
			})
		}
	}
	return out
}
