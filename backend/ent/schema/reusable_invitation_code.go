package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ReusableInvitationCode holds reusable registration gate codes.
type ReusableInvitationCode struct {
	ent.Schema
}

func (ReusableInvitationCode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "reusable_invitation_codes"},
	}
}

func (ReusableInvitationCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(64).NotEmpty().Unique(),
		field.String("status").MaxLen(20).Default("active"),
		field.Int("max_uses").Default(0).NonNegative(),
		field.Int("used_count").Default(0).NonNegative(),
		field.Time("expires_at").Optional().Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Time("created_at").Immutable().Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ReusableInvitationCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("uses", ReusableInvitationCodeUse.Type),
	}
}

func (ReusableInvitationCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("expires_at"),
	}
}
