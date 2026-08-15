package schema

import (
	"fmt"

	"github.com/LuckyKuang/sub2api-plus/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Team is a single-owner billing boundary for one layer of members.
type Team struct {
	ent.Schema
}

func (Team) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "teams"}}
}

func (Team) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}, mixins.SoftDeleteMixin{}}
}

func (Team) Fields() []ent.Field {
	validateDefaultLimit := func(value float64) error {
		if value < 0 {
			return fmt.Errorf("team member default limit cannot be negative")
		}
		return nil
	}
	defaultLimit := func(name string) ent.Field {
		return field.Float(name).
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Validate(validateDefaultLimit)
	}
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("status").MaxLen(20).Default("active").Validate(func(value string) error {
			if value != "active" && value != "suspended" {
				return fmt.Errorf("team status must be active or suspended")
			}
			return nil
		}),
		field.Int("member_limit").Default(10).NonNegative(),
		defaultLimit("default_daily_limit_usd"),
		defaultLimit("default_weekly_limit_usd"),
		defaultLimit("default_monthly_limit_usd"),
	}
}

func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("memberships", TeamMembership.Type),
		edge.To("invitations", TeamInvitation.Type),
		edge.To("ownership_transfers", TeamOwnershipTransfer.Type),
		edge.To("api_keys", APIKey.Type),
		edge.To("usage_logs", UsageLog.Type),
	}
}

func (Team) Indexes() []ent.Index {
	return []ent.Index{index.Fields("status"), index.Fields("deleted_at")}
}
