package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RPLogin struct{ ent.Schema }

func (RPLogin) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("browser_hash").Immutable(),
		field.Bytes("payload").Sensitive(),
		field.Time("expires_at"),
		field.Bool("consumed").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (RPLogin) Indexes() []ent.Index { return []ent.Index{index.Fields("expires_at")} }
func (RPLogin) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "sessions", Table: "rp_logins"}}
}
