package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type LogoutInteraction struct{ ent.Schema }

func (LogoutInteraction) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("client_id").Immutable(), field.String("browser_hash").Sensitive().Immutable(), field.Bytes("payload").Sensitive(), field.Bool("consumed").Default(false), field.Time("expires_at").Immutable()}
}
func (LogoutInteraction) Indexes() []ent.Index { return []ent.Index{index.Fields("expires_at")} }
func (LogoutInteraction) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "logout_interactions"}}
}
