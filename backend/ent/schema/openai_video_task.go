package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OpenAIVideoTask is the durable routing and billing source for one accepted
// OpenAI-compatible asynchronous video request.
type OpenAIVideoTask struct{ ent.Schema }

func (OpenAIVideoTask) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "openai_video_tasks"}}
}

func (OpenAIVideoTask) Fields() []ent.Field {
	decimal := map[string]string{dialect.Postgres: "decimal(20,10)"}
	timestamptz := map[string]string{dialect.Postgres: "timestamptz"}
	return []ent.Field{
		field.String("local_request_id").MaxLen(128).Immutable(),
		field.String("task_id").MaxLen(255).Optional().Nillable(),
		field.Int64("actor_user_id"), field.Int64("billing_user_id"),
		field.Int64("team_id").Optional().Nillable(), field.Int64("api_key_id"),
		field.Int64("group_id"), field.Int64("channel_id").Optional().Nillable(),
		field.Int64("account_id"), field.Int64("subscription_id").Optional().Nillable(),
		field.String("requested_model").MaxLen(128), field.String("upstream_model").MaxLen(128),
		field.Int("request_seconds"), field.String("resolution").MaxLen(16),
		field.String("status").MaxLen(32).Default("creating"),
		field.String("upstream_status").MaxLen(64).Optional().Nillable(),
		field.Int8("billing_type"), field.String("billing_status").MaxLen(32).Default("none"),
		field.Float("total_cost").SchemaType(decimal).Default(0),
		field.Float("actual_cost").SchemaType(decimal).Optional().Nillable(),
		field.Float("hold_amount").SchemaType(decimal).Default(0),
		field.Float("group_rate_multiplier").SchemaType(decimal).Default(1),
		field.Float("account_rate_multiplier").SchemaType(decimal).Default(1),
		field.Bool("allowance_reserved").Default(false),
		field.String("request_payload_hash").MaxLen(64),
		field.String("inbound_endpoint").MaxLen(255), field.String("upstream_endpoint").MaxLen(255),
		field.String("model_mapping_chain").MaxLen(512).Optional().Nillable(),
		field.String("user_agent").SchemaType(map[string]string{dialect.Postgres: "text"}).Optional().Nillable(),
		field.String("ip_address").MaxLen(64).Optional().Nillable(),
		field.Int("retry_count").Default(0),
		field.Time("next_poll_at").SchemaType(timestamptz).Optional().Nillable(),
		field.Time("lease_until").SchemaType(timestamptz).Optional().Nillable(),
		field.String("lease_token").MaxLen(128).Optional().Nillable(),
		field.String("last_error_code").MaxLen(128).Optional().Nillable(),
		field.String("last_error_message").SchemaType(map[string]string{dialect.Postgres: "text"}).Optional().Nillable(),
		field.Bool("usage_recorded").Default(false),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(timestamptz),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now).SchemaType(timestamptz),
		field.Time("submitted_at").SchemaType(timestamptz).Optional().Nillable(),
		field.Time("finished_at").SchemaType(timestamptz).Optional().Nillable(),
		field.Time("settled_at").SchemaType(timestamptz).Optional().Nillable(),
		field.Time("usage_recorded_at").SchemaType(timestamptz).Optional().Nillable(),
	}
}

func (OpenAIVideoTask) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("local_request_id").Unique(),
		index.Fields("task_id").Unique().Annotations(entsql.IndexWhere("task_id IS NOT NULL")),
		index.Fields("api_key_id", "created_at"),
		index.Fields("billing_status", "updated_at"),
	}
}
