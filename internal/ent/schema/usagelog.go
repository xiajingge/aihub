package schema

import "entgo.io/ent"

// UsageLog holds the schema definition for the UsageLog entity.
type UsageLog struct {
	ent.Schema
}

// Fields of the UsageLog.
func (UsageLog) Fields() []ent.Field {
	return nil
}

// Edges of the UsageLog.
func (UsageLog) Edges() []ent.Edge {
	return nil
}
