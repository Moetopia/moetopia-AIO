package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type ClientTestSubject struct{ ent.Schema }

func (ClientTestSubject) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("client_id").Immutable(), field.String("user_id").Immutable(), field.Time("created_at").Default(time.Now).Immutable()}
}
func (ClientTestSubject) Indexes() []ent.Index {
	return []ent.Index{index.Fields("client_id", "user_id").Unique(), index.Fields("user_id")}
}
func (ClientTestSubject) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "client_test_subjects"}}
}
