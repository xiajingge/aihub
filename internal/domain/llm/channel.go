package llm

import (
	"fmt"
	"slices"

	"github.com/xiajignge/aihub/internal/domain/transformer"
	"github.com/xiajignge/aihub/internal/ent"
)

type Channel struct {
	*ent.Channel

	// Outbound 是该通道对应的出站转换器。
	Outbound transformer.Outbound
}

// ChooseModel 选择该通道实际使用的模型（支持模型映射）。
func (c Channel) ChooseModel(model string) (string, error) {
	if slices.Contains(c.SupportedModels, model) {
		return model, nil
	}

	if c.Settings == nil {
		return "", fmt.Errorf("model %s not supported in channel %s", model, c.Name)
	}

	for _, mapping := range c.Settings.ModelMappings {
		if mapping.From == model && slices.Contains(c.SupportedModels, mapping.To) {
			return mapping.To, nil
		}
	}

	return "", fmt.Errorf("model %s not supported in channel %s", model, c.Name)
}

// IsModelSupported 判断该通道是否支持给定模型（含映射后的目标模型）。
func (c Channel) IsModelSupported(model string) bool {
	if slices.Contains(c.SupportedModels, model) {
		return true
	}

	if c.Settings == nil {
		return false
	}

	for _, mapping := range c.Settings.ModelMappings {
		if mapping.From == model && slices.Contains(c.SupportedModels, mapping.To) {
			return true
		}
	}

	return false
}
