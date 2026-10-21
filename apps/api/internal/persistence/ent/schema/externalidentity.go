package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type ExternalIdentity struct{ ent.Schema }

func (ExternalIdentity) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("user_id").Immutable(),
		field.Enum("provider").Values("github", "google").Immutable(),
		field.String("issuer").Immutable(),
		field.String("subject").Sensitive().Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (ExternalIdentity) Indexes() []ent.Index {
	return []ent.Index{index.Fields("provider", "issuer", "subject").Unique(), index.Fields("user_id", "provider").Unique()}
}
func (ExternalIdentity) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "identity", Table: "external_identities"}}
}
