package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type SecurityEvent struct{ ent.Schema }

func (SecurityEvent) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("user_id").Immutable(), field.String("action").Immutable(), field.Time("created_at").Default(time.Now).Immutable()}
}
func (SecurityEvent) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "created_at"), index.Fields("created_at")}
}
func (SecurityEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "audit", Table: "security_events"}}
}
