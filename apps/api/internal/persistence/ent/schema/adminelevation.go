package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AdminElevation struct{ ent.Schema }

func (AdminElevation) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("user_id").Immutable(), field.Int("factor_version"), field.Time("expires_at")}
}
func (AdminElevation) Indexes() []ent.Index {
	return []ent.Index{index.Fields("expires_at"), index.Fields("user_id")}
}
func (AdminElevation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "administration", Table: "elevations"}}
}
