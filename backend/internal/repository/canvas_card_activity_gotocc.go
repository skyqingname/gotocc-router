package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// Order IDs are negative only in the activity projection so they cannot collide
// with the original positive redeem-code IDs. The order keeps its real ID.
const redeemActivityRows = `
 SELECT rc.id,rc.code,rc.type,rc.value,rc.status,rc.used_at,rc.created_at,
   COALESCE(rc.notes,''),rc.group_id,rc.validity_days,g.name,
   NULL::bigint AS order_id,NULL::uuid AS operation_id,NULL::integer AS denomination,
   NULL::text AS funding_source,NULL::bigint AS team_id,NULL::numeric AS balance_after,
   NULL::text AS card_code,g.i18n
 FROM redeem_codes rc LEFT JOIN groups g ON g.id=rc.group_id
 WHERE rc.used_by=$1 AND rc.type<>'canvas_card'
 UNION ALL
 SELECT -o.id,'','canvas_card',-o.denomination,'used',o.created_at,o.created_at,
   '',NULL,0,NULL,o.id,o.operation_id,o.denomination,o.funding_source,o.team_id,o.balance_after,i.code,NULL::jsonb
 FROM canvas_card_orders o JOIN canvas_card_inventory i ON i.issued_order_id=o.id
 WHERE o.user_id=$1`

func (r *canvasCardRepository) Activity(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.RedeemActivity, int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT
   (SELECT COUNT(*) FROM redeem_codes WHERE used_by=$1 AND type<>'canvas_card')+
   (SELECT COUNT(*) FROM canvas_card_orders WHERE user_id=$1)`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM (`+redeemActivityRows+`) activity ORDER BY used_at DESC NULLS LAST,id DESC LIMIT $2 OFFSET $3`, userID, params.Limit(), params.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]service.RedeemActivity, 0)
	for rows.Next() {
		var item service.RedeemActivity
		var groupName, operationID, fundingSource, cardCode sql.NullString
		var orderID, denomination, teamID sql.NullInt64
		var balanceAfter sql.NullFloat64
		var groupI18n []byte
		if err := rows.Scan(&item.ID, &item.Code, &item.Type, &item.Value, &item.Status, &item.UsedAt, &item.CreatedAt,
			&item.Notes, &item.GroupID, &item.ValidityDays, &groupName, &orderID, &operationID, &denomination, &fundingSource, &teamID, &balanceAfter, &cardCode, &groupI18n); err != nil {
			return nil, 0, err
		}
		if groupName.Valid && item.GroupID != nil {
			item.Group = &service.RedeemActivityGroup{ID: *item.GroupID, Name: groupName.String}
			if err := json.Unmarshal(groupI18n, &item.Group.I18n); err != nil {
				return nil, 0, err
			}
		}
		if orderID.Valid {
			item.CanvasCard = &service.CanvasCardOrder{ID: orderID.Int64, OperationID: operationID.String, UserID: userID,
				Denomination: int(denomination.Int64), FundingSource: fundingSource.String, BalanceAfter: balanceAfter.Float64, Code: cardCode.String, CreatedAt: item.CreatedAt}
			if teamID.Valid {
				item.CanvasCard.TeamID = &teamID.Int64
			}
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}
