package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type AuthorizationGrant struct{ ent.Schema }

func (AuthorizationGrant) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("client_id").Immutable(),
		field.String("user_id").Default(""),
		field.Int("security_version").Default(0),
		field.Strings("scopes").Default([]string{}),
		field.Strings("audiences").Default([]string{}),
		field.Time("expires_at"),
		field.Bool("revoked").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (AuthorizationGrant) Indexes() []ent.Index {
	return []ent.Index{index.Fields("expires_at"), index.Fields("client_id"), index.Fields("user_id", "created_at")}
}
func (AuthorizationGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "grants"}}
}
