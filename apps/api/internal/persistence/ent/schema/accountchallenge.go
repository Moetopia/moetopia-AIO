package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type AccountChallenge struct{ ent.Schema }

func (AccountChallenge) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.Enum("purpose").Values("reset", "email").Immutable(),
		field.String("user_id").Immutable(),
		field.String("email").Immutable(),
		field.String("binding").Immutable(),
		field.Int("security_version").Immutable(),
		field.Bytes("code_hash").Sensitive().Immutable(),
		field.Int("attempts").Default(0),
		field.Bool("consumed").Default(false),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (AccountChallenge) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id"), index.Fields("expires_at")}
}
func (AccountChallenge) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "account_challenges"}}
}
