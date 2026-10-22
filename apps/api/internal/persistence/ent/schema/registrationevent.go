package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RegistrationEvent struct{ ent.Schema }

func (RegistrationEvent) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("day").Immutable()}
}
func (RegistrationEvent) Indexes() []ent.Index { return []ent.Index{index.Fields("day")} }
func (RegistrationEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "registration_events"}}
}
