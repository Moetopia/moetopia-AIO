package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type AccountUser struct{ ent.Schema }

func (AccountUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.String("email").Unique(),
		field.String("display_name").NotEmpty(),
		field.String("password_hash").Sensitive(),
		field.Bool("enabled").Default(true),
		field.Time("email_verified_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (AccountUser) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "users"}}
}
