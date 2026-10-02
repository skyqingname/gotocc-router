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

// ReusableInvitationCodeUse records successful reusable-code signups.
type ReusableInvitationCodeUse struct {
	ent.Schema
}

func (ReusableInvitationCodeUse) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "reusable_invitation_code_uses"},
	}
}

func (ReusableInvitationCodeUse) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("code_id"),
		field.Int64("user_id"),
		field.String("email").MaxLen(255).Default(""),
		field.String("auth_source").MaxLen(50).Default(""),
		field.Time("used_at").Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ReusableInvitationCodeUse) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("code", ReusableInvitationCode.Type).
			Ref("uses").Field("code_id").Required().Unique(),
		edge.From("user", User.Type).
			Ref("reusable_invitation_code_uses").Field("user_id").Required().Unique(),
	}
}

func (ReusableInvitationCodeUse) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code_id", "used_at"),
		index.Fields("user_id"),
	}
}
