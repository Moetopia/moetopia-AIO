package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type AdminControl struct{ ent.Schema }

func (AdminControl) Fields() []ent.Field { return []ent.Field{field.String("id").Immutable()} }

func (AdminControl) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "administration", Table: "controls"}}
}
