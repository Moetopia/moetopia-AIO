package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type SigningKey struct{ ent.Schema }

func (SigningKey) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("algorithm").Default("RS256"),
		field.Bytes("private_key").Sensitive(),
		field.Bytes("public_key"),
		field.Bool("active").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("retire_after").Optional().Nillable(),
	}
}

func (SigningKey) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "signing_keys"}}
}
