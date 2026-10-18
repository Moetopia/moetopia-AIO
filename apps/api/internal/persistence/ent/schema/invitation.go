package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Invitation struct{ ent.Schema }

func (Invitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.Time("expires_at"),
		field.Bool("consumed").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Invitation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "invitations"}}
}
