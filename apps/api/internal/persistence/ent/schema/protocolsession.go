package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProtocolSession struct{ ent.Schema }

func (ProtocolSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable(),
		field.String("kind"),
		field.String("request_id"),
		field.String("client_id"),
		field.Bytes("payload").Sensitive(),
		field.Time("expires_at"),
		field.Bool("revoked").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (ProtocolSession) Indexes() []ent.Index {
	return []ent.Index{index.Fields("request_id", "kind"), index.Fields("expires_at")}
}

func (ProtocolSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Schema: "authorization", Table: "protocol_sessions"}}
}
