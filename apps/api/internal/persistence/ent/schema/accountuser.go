package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type AccountUser struct{ ent.Schema }

func (AccountUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.String("email").Unique(),
		field.String("display_name").NotEmpty(),
		field.String("bio").Default(""),
		field.Bytes("avatar").Optional(),
		field.String("avatar_digest").Default(""),
		field.Int("revision").Default(0),
		field.Int("security_version").Default(0),
		field.String("password_hash").Sensitive(),
		field.Bool("enabled").Default(true),
		field.String("disabled_reason").Default(""),
		field.Time("deletion_requested_at").Optional().Nillable(),
		field.Time("deletion_due_at").Optional().Nillable(),
		field.Time("email_verified_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (AccountUser) Indexes() []ent.Index {
	return []ent.Index{index.Fields("deletion_due_at")}
}

func (AccountUser) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "users"}}
}
