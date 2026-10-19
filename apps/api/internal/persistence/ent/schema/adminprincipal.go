package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type AdminPrincipal struct{ ent.Schema }

func (AdminPrincipal) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(), field.Bool("enabled").Default(true),
		field.Bytes("secret").Optional().Sensitive(), field.Bytes("pending_secret").Optional().Sensitive(),
		field.String("pending_session").Default(""), field.Time("pending_expires_at").Optional().Nillable(),
		field.Int64("last_counter").Default(-1), field.Int("factor_version").Default(0),
		field.Strings("recovery_hashes").Default([]string{}).Sensitive(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (AdminPrincipal) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "administration", Table: "principals"}}
}
