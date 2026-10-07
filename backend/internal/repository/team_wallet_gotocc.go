package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

func NewTeamWalletRepository(db *sql.DB) service.TeamWalletRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) GetWallet(ctx context.Context, teamID int64) (*service.TeamWallet, error) {
	wallet := &service.TeamWallet{}
	err := r.db.QueryRowContext(ctx, `SELECT balance, frozen_balance, reseller_owner_id FROM teams WHERE id=$1`, teamID).
		Scan(&wallet.Balance, &wallet.FrozenBalance, &wallet.ResellerOwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrTeamNotFound
	}
	return wallet, err
}

// Wallet settlement and owner controls take the team row before membership rows.
func lockTeamRow(ctx context.Context, tx *sql.Tx, teamID int64) error {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM teams WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, teamID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrTeamNotFound
	}
	return err
}

func lockTeamOwner(ctx context.Context, tx *sql.Tx, teamID, userID int64) error {
	if err := lockTeamRow(ctx, tx, teamID); err != nil {
		return err
	}
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM team_memberships WHERE team_id=$1 AND user_id=$2 AND role='owner' AND left_at IS NULL FOR UPDATE`, teamID, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrTeamOwnerRequired
	}
	return err
}

func checkTeamTransferSource(ctx context.Context, tx *sql.Tx, teamID, targetUserID int64) error {
	var source, targetSource sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT t.reseller_owner_id,c.owner_user_id FROM teams t
		LEFT JOIN reseller_customers c ON c.user_id=$2 WHERE t.id=$1 AND t.deleted_at IS NULL FOR SHARE OF t`, teamID, targetUserID).Scan(&source, &targetSource)
	if err != nil {
		return err
	}
	if source != targetSource {
		return service.ErrTeamFundingSource
	}
	return nil
}

func (r *teamRepository) ownerUpdate(ctx context.Context, teamID int64, query string, args ...any) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockTeamOwner(ctx, tx, teamID, service.TeamOwnerMutationUserID(ctx)); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err = requireTeamAffected(result, err); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *teamRepository) FundWallet(ctx context.Context, teamID, ownerID int64, operationID string, amount float64) (*service.TeamFundingResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = lockTeamOwner(ctx, tx, teamID, ownerID); err != nil {
		return nil, err
	}
	var source sql.NullInt64
	var balance, frozen float64
	err = tx.QueryRowContext(ctx, `SELECT balance,frozen_balance,reseller_owner_id FROM teams WHERE id=$1 AND deleted_at IS NULL AND status='active' FOR UPDATE`, teamID).Scan(&balance, &frozen, &source)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrTeamSuspended
	}
	if err != nil {
		return nil, err
	}
	result := &service.TeamFundingResult{OperationID: operationID, Amount: amount}
	var previousActor int64
	err = tx.QueryRowContext(ctx, `SELECT actor_user_id,amount,personal_balance_after,balance_after FROM team_wallet_entries WHERE team_id=$1 AND operation_id=$2 AND api_key_id=0`, teamID, operationID).
		Scan(&previousActor, &result.Amount, &result.PersonalBalance, &result.TeamBalance)
	if err == nil {
		if previousActor != ownerID || result.Amount != amount {
			return nil, service.ErrTeamFundingConflict
		}
		return result, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var personalSource sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT owner_user_id FROM reseller_customers WHERE user_id=$1 FOR UPDATE`, ownerID).Scan(&personalSource)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if source != personalSource {
		return nil, service.ErrTeamFundingSource
	}
	if source.Valid {
		var personalFrozen float64
		err = tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance-$2 WHERE user_id=$1 AND credit_balance >= $2 RETURNING credit_balance,frozen_credit`, ownerID, amount).Scan(&result.PersonalBalance, &personalFrozen)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrResellerCreditInsufficient
		}
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO reseller_credit_entries (customer_user_id,owner_user_id,operator_user_id,operation_id,kind,amount,balance_after,frozen_after,notes)
			VALUES($1,$2,$1,$3,'team_transfer',-$4,$5,$6,$7)`, ownerID, source.Int64, "team:"+operationID, amount, result.PersonalBalance, personalFrozen, fmt.Sprintf("转入团队 %d 公共余额", teamID))
	} else {
		err = tx.QueryRowContext(ctx, `UPDATE users SET balance=balance-$2,updated_at=NOW() WHERE id=$1 AND balance >= $2 AND status='active' AND deleted_at IS NULL RETURNING balance`, ownerID, amount).Scan(&result.PersonalBalance)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrInsufficientBalance
		}
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO redeem_codes(code,type,value,status,used_by,used_at,notes) VALUES($1,'team_transfer',-$2,'used',$3,NOW(),$4)`, "team:"+operationID, amount, ownerID, fmt.Sprintf("转入团队 %d 公共余额", teamID))
	}
	if err != nil {
		return nil, err
	}
	err = tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance+$2,updated_at=NOW() WHERE id=$1 RETURNING balance`, teamID, amount).Scan(&result.TeamBalance)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO team_wallet_entries(team_id,operation_id,kind,actor_user_id,amount,balance_after,frozen_after,personal_balance_after)
		VALUES($1,$2,'fund',$3,$4,$5,$6,$7)`, teamID, operationID, ownerID, amount, result.TeamBalance, frozen, result.PersonalBalance)
	if err != nil {
		return nil, err
	}
	return result, tx.Commit()
}
