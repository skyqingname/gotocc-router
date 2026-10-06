package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

func (r *resellerRepository) CustomerAccount(ctx context.Context, userID int64) (*service.ResellerCustomerAccount, error) {
	account := &service.ResellerCustomerAccount{}
	err := scanSingleRow(ctx, r.executor(ctx), `SELECT c.user_id,c.owner_user_id,u.username,
		c.credit_balance::float8,c.frozen_credit::float8,u.balance::float8,
		(u.status='active' AND u.deleted_at IS NULL),(customer.status='active' AND customer.deleted_at IS NULL),
        CASE WHEN p.enabled AND p.contact_enabled THEN p.contact_info ELSE '' END,
        COALESCE(p.enabled AND p.announcements_enabled,FALSE)
		FROM reseller_customers c JOIN users customer ON customer.id=c.user_id JOIN users u ON u.id=c.owner_user_id
        LEFT JOIN reseller_profiles p ON p.user_id=c.owner_user_id WHERE c.user_id=$1`,
		[]any{userID}, &account.UserID, &account.OwnerID, &account.OwnerName,
		&account.CreditBalance, &account.FrozenCredit, &account.OwnerBalance, &account.OwnerActive, &account.CustomerActive, &account.ContactInfo, &account.AnnouncementsEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (r *resellerRepository) CustomerTransaction(ctx context.Context, ownerID, customerID int64, apply func(context.Context) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	txctx := dbent.NewTxContext(ctx, tx)
	q := r.executor(txctx)
	var id int64
	if err = scanSingleRow(txctx, q, `SELECT user_id FROM reseller_profiles WHERE user_id=$1 AND enabled FOR SHARE`, []any{ownerID}, &id); err != nil {
		return err
	}
	if customerID != 0 {
		if err = scanSingleRow(txctx, q, `SELECT user_id FROM reseller_customers WHERE user_id=$1 AND owner_user_id=$2 FOR UPDATE`, []any{customerID, ownerID}, &id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return service.ErrResellerCustomerNotFound
			}
			return err
		}
	}
	if err = apply(txctx); err != nil {
		return err
	}
	return tx.Commit()
}

const resellerCreditColumns = `id,customer_user_id,owner_user_id,operation_id,kind,amount::float8,frozen_amount::float8,balance_after::float8,frozen_after::float8,platform_cost::float8,model,notes,created_at,deduct_all`

func scanResellerCreditEntry(row interface{ Scan(...any) error }) (*service.ResellerCreditEntry, error) {
	e := &service.ResellerCreditEntry{}
	err := row.Scan(&e.ID, &e.CustomerID, &e.OwnerID, &e.OperationID, &e.Kind, &e.Amount, &e.FrozenAmount, &e.BalanceAfter, &e.FrozenAfter, &e.PlatformCost, &e.Model, &e.Notes, &e.CreatedAt, &e.DeductAll)
	return e, err
}

func (r *resellerRepository) ChangeCustomerCredit(ctx context.Context, ownerID, customerID int64, input service.ResellerCreditInput) (*service.ResellerCreditEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var balance, frozen float64
	err = tx.QueryRowContext(ctx, `SELECT credit_balance::float8,frozen_credit::float8 FROM reseller_customers WHERE user_id=$1 AND owner_user_id=$2 FOR UPDATE`, customerID, ownerID).Scan(&balance, &frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrResellerCustomerNotFound
	}
	if err != nil {
		return nil, err
	}
	delta := input.Amount
	if input.Kind == "deduct" {
		delta = -delta
	}
	previous, err := scanResellerCreditEntry(tx.QueryRowContext(ctx, `SELECT `+resellerCreditColumns+` FROM reseller_credit_entries WHERE customer_user_id=$1 AND operation_id=$2 AND api_key_id=0`, customerID, input.OperationID))
	if err == nil {
		if previous.Kind != input.Kind || previous.DeductAll != input.DeductAll || (!input.DeductAll && previous.Amount != delta) || previous.Notes != input.Notes {
			return nil, infraerrors.Conflict("CUSTOMER_CREDIT_OPERATION_CONFLICT", "这笔额度操作已经使用不同内容提交")
		}
		return previous, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if input.DeductAll {
		if balance <= 0 {
			return nil, infraerrors.BadRequest("NO_AVAILABLE_CUSTOMER_CREDIT", "没有可减少的额度")
		}
		delta = -balance
	}
	if balance+delta < 0 {
		return nil, service.ErrResellerCreditInsufficient
	}
	err = tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance+$3 WHERE user_id=$1 AND owner_user_id=$2 RETURNING credit_balance::float8`, customerID, ownerID, delta).Scan(&balance)
	if err != nil {
		return nil, err
	}
	entry, err := scanResellerCreditEntry(tx.QueryRowContext(ctx, `INSERT INTO reseller_credit_entries
		(customer_user_id,owner_user_id,operator_user_id,operation_id,kind,amount,balance_after,frozen_after,notes,deduct_all)
		VALUES($1,$2,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+resellerCreditColumns,
		customerID, ownerID, input.OperationID, input.Kind, delta, balance, frozen, input.Notes, input.DeductAll))
	if err != nil {
		return nil, err
	}
	return entry, tx.Commit()
}

func (r *resellerRepository) CustomerCreditEntries(ctx context.Context, ownerID, customerID int64, page, size int) ([]service.ResellerCreditEntry, int64, error) {
	q := r.executor(ctx)
	var count int64
	if err := scanSingleRow(ctx, q, `SELECT COUNT(*) FROM reseller_credit_entries WHERE owner_user_id=$1 AND customer_user_id=$2`, []any{ownerID, customerID}, &count); err != nil {
		return nil, 0, err
	}
	rows, err := q.QueryContext(ctx, `SELECT `+resellerCreditColumns+` FROM reseller_credit_entries WHERE owner_user_id=$1 AND customer_user_id=$2 ORDER BY created_at DESC,id DESC LIMIT $3 OFFSET $4`, ownerID, customerID, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	entries := []service.ResellerCreditEntry{}
	for rows.Next() {
		entry, e := scanResellerCreditEntry(rows)
		if e != nil {
			return nil, 0, e
		}
		entries = append(entries, *entry)
	}
	return entries, count, rows.Err()
}
