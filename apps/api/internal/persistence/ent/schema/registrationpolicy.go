package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type RegistrationPolicy struct{ ent.Schema }

func (RegistrationPolicy) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.Enum("mode").Values("invite", "open").Default("invite")}
}
func (RegistrationPolicy) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "registration_policy"}}
}
