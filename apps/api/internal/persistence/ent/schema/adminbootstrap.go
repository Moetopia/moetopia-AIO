package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type AdminBootstrap struct{ ent.Schema }

func (AdminBootstrap) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("user_id").Immutable(), field.Time("created_at").Default(time.Now).Immutable()}
}
func (AdminBootstrap) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "administration", Table: "bootstrap"}}
}
