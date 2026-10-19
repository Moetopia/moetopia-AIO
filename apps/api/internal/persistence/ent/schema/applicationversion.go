package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type ApplicationVersion struct{ ent.Schema }

func (ApplicationVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(), field.String("application_id").Immutable(), field.Int("version").Immutable(),
		field.JSON("config", map[string]any{}), field.Enum("status").Values("draft", "pending", "approved", "rejected").Default("draft"),
		field.String("review_reason").Default(""), field.String("reviewer_id").Default(""), field.Time("created_at").Default(time.Now).Immutable(),
	}
}
func (ApplicationVersion) Indexes() []ent.Index {
	return []ent.Index{index.Fields("application_id", "version").Unique(), index.Fields("status", "created_at")}
}
func (ApplicationVersion) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "developer", Table: "application_versions"}}
}
