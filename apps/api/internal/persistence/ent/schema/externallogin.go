package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ExternalLogin struct{ ent.Schema }

func (ExternalLogin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("browser_hash").Sensitive().Immutable(),
		field.Enum("provider").Values("github", "google").Immutable(),
		field.Enum("purpose").Values("login", "bind", "register").Immutable(),
		field.String("user_id").Default("").Immutable(),
		field.Bytes("payload").Sensitive(),
		field.Bool("consumed").Default(false),
		field.Time("expires_at").Immutable(),
	}
}
func (ExternalLogin) Indexes() []ent.Index {
	return []ent.Index{index.Fields("expires_at"), index.Fields("user_id")}
}
func (ExternalLogin) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "external_logins"}}
}
