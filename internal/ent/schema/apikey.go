package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/xiajignge/aihub/internal/objects"
)

// APIKey holds the schema definition for the APIKey entity.
type APIKey struct {
	ent.Schema
}

func (APIKey) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		SoftDeleteMixin{},
	}
}

func (APIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").
			StorageKey("api_keys_by_user_id"),
		index.Fields("key").
			StorageKey("api_keys_by_key").
			Unique(),
	}
}

func (APIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").Immutable().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.String("key").Immutable().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
		field.String("name"),
		field.Enum("status").Values("enabled", "disabled").Default("enabled").Annotations(
			entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
		),
		field.Strings("scopes").
			Comment("API Key specific scopes: read_channels, write_requests, etc.").
			Default([]string{"read_channels", "write_requests"}).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			).
			Sensitive().
			Optional(),
		field.JSON("profiles", &objects.APIKeyProfiles{}).
			Default(&objects.APIKeyProfiles{}).
			Optional().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			),
	}
}

func (APIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Unique().
			Immutable().
			Required().
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
			).
			Ref("api_keys").Field("user_id"),
		edge.To("requests", Request.Type).
			Annotations(
				entgql.Skip(entgql.SkipMutationCreateInput, entgql.SkipMutationUpdateInput),
				entgql.RelayConnection(),
			),
	}
}
