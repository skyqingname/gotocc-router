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

// ImageObject stores durable ownership for images offloaded by async Images
// tasks. Object bytes remain in S3/R2; this table stores only references.
type ImageObject struct {
	ent.Schema
}

func (ImageObject) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "image_objects"}}
}

func (ImageObject) Fields() []ent.Field {
	return []ent.Field{
		field.String("object_id").MaxLen(64).NotEmpty().Immutable().Unique(),
		field.Int64("user_id").Immutable(),
		field.Int64("api_key_id").Immutable(),
		field.String("task_id").MaxLen(64).NotEmpty().Immutable(),
		field.String("storage_key").MaxLen(1024).NotEmpty().Immutable().Unique(),
		field.String("content_type").MaxLen(128).Default("image/png"),
		field.Int64("byte_size").NonNegative(),
		field.Time("created_at").Immutable().Default(time.Now).SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ImageObject) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("task_id"),
		index.Fields("api_key_id", "created_at"),
	}
}
