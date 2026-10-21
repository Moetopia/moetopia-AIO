package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OAuthClient struct{ ent.Schema }

func (OAuthClient) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty().Immutable(),
		field.Bytes("secret_hash").Sensitive().Optional(),
		field.String("name").NotEmpty(),
		field.String("owner_id").Default(""),
		field.Bool("test_only").Default(false),
		field.Bool("public").Default(false),
		field.Bool("enabled").Default(false),
		field.Strings("redirect_uris").Default([]string{}),
		field.Strings("grant_types").Default([]string{}),
		field.Strings("response_types").Default([]string{}),
		field.Strings("scopes").Default([]string{}),
		field.Strings("audiences").Default([]string{}),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (OAuthClient) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "clients"}}
}

func (OAuthClient) Indexes() []ent.Index {
	return []ent.Index{index.Fields("owner_id")}
}
