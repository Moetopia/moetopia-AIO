package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EmailDelivery struct{ ent.Schema }

func (EmailDelivery) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.Bytes("payload").Sensitive(),
		field.Int("attempts").Default(0),
		field.Time("available_at"),
		field.Time("lease_until"),
		field.Time("expires_at"),
	}
}
func (EmailDelivery) Indexes() []ent.Index {
	return []ent.Index{index.Fields("available_at", "lease_until")}
}
func (EmailDelivery) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "notifications", Table: "email_deliveries"}}
}
