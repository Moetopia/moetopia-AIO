package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type RegistrationStatisticsState struct{ ent.Schema }

func (RegistrationStatisticsState) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.Time("started_at").Default(time.Now).Immutable()}
}
func (RegistrationStatisticsState) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "registration_statistics_state"}}
}
