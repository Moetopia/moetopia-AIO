package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProtocolEvent struct{ ent.Schema }

func (ProtocolEvent) Fields() []ent.Field {
	return []ent.Field{field.String("id").Immutable(), field.String("client_id").Immutable(), field.String("day").Immutable(), field.Enum("kind").Values("authorization", "code_exchange", "refresh", "machine_token").Immutable()}
}

func (ProtocolEvent) Indexes() []ent.Index {
	return []ent.Index{index.Fields("client_id", "day", "kind"), index.Fields("day")}
}

func (ProtocolEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "protocol_events"}}
}
