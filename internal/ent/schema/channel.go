package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/xiajignge/aihub/internal/objects"
)

type Channel struct {
	ent.Schema
}

func (Channel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		SoftDeleteMixin{},
	}
}

func (Channel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").
			StorageKey("channels_by_name").
			Unique(),
	}
}

func (Channel) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("type").
			Values(
				"openai",
				"anthropic",
				"anthropic_aws",
				"anthropic_gcp",
				// "gemini",
				"deepseek",
				"deepseek_anthropic",
				"doubao",
				"moonshot",
				"moonshot_anthropic",
				"zhipu",
				"zai",
				"zhipu_anthropic",
				"zai_anthropic",
				"anthropic_fake",
				"openai_fake",
			).
			Immutable(),
		field.String("base_url").Optional(),
		field.String("name"),
		field.Enum("status").Values("enabled", "disabled", "archived").Default("disabled"),
		field.JSON("credentials", &objects.ChannelCredentials{}).Sensitive().Default(&objects.ChannelCredentials{}),
		field.Strings("supported_models"),
		field.String("default_test_model"),
		field.JSON("settings", &objects.ChannelSettings{}).
			Default(&objects.ChannelSettings{
				ModelMappings: []objects.ModelMapping{},
			}).Optional(),
		field.Int("ordering_weight").Default(0).Comment("Ordering weight for display sorting"),
	}
}

func (Channel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("requests", Request.Type),
		edge.To("executions", RequestExecution.Type),
		edge.To("usage_logs", UsageLog.Type),
	}
}
