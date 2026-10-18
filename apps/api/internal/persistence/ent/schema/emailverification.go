package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type EmailVerification struct{ ent.Schema }

func (EmailVerification) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("email"),
		field.Bytes("code_hash").Sensitive(),
		field.Int("attempts").Default(0),
		field.Bool("consumed").Default(false),
		field.Time("expires_at"),
	}
}

func (EmailVerification) Indexes() []ent.Index { return []ent.Index{index.Fields("expires_at")} }
func (EmailVerification) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "email_verifications"}}
}
