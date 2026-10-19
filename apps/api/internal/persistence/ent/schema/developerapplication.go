package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type DeveloperApplication struct{ ent.Schema }

func (DeveloperApplication) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(), field.String("owner_id").Immutable(), field.String("client_id").Unique().Immutable(),
		field.Bool("public").Immutable(), field.Bool("enabled").Default(true), field.Bool("deleted").Default(false),
		field.Int("revision").Default(1), field.Int("active_version").Default(0), field.Int("latest_version").Default(1),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (DeveloperApplication) Indexes() []ent.Index {
	return []ent.Index{index.Fields("owner_id", "created_at")}
}
func (DeveloperApplication) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "developer", Table: "applications"}}
}
