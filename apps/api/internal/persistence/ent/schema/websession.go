package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type WebSession struct{ ent.Schema }

func (WebSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("user_id").Immutable(),
		field.String("application").Immutable(),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (WebSession) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id"), index.Fields("expires_at")}
}
func (WebSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "sessions", Table: "web_sessions"}}
}
