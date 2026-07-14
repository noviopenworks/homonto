package opencode

import (
	"testing"

	"github.com/noviopenworks/homonto/internal/config"
)

func TestDesiredSettings_ModelRoutesProjectDefaults(t *testing.T) {
	c := &config.Config{
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "openai/gpt-5.6-terra"},
				"coding":        {Model: "openai/gpt-5.6-mini"},
				"trivial":       {Model: "openai/gpt-5.6-nano"},
			},
		},
	}
	got := desiredSettings(c)
	if got["setting.model"] != `"openai/gpt-5.6-terra"` {
		t.Errorf("architectural route must project setting.model, got %q", got["setting.model"])
	}
	if got["setting.small_model"] != `"openai/gpt-5.6-nano"` {
		t.Errorf("trivial route must project setting.small_model, got %q", got["setting.small_model"])
	}
}

func TestDesiredSettings_ExplicitSettingWinsOverRoute(t *testing.T) {
	c := &config.Config{
		Settings: config.Settings{OpenCode: map[string]any{"model": "explicit/model"}},
		Models: config.ModelConfig{
			OpenCode: map[string]config.ModelRoute{
				"architectural": {Model: "route/model"},
			},
		},
	}
	if got := desiredSettings(c); got["setting.model"] != `"explicit/model"` {
		t.Errorf("explicit [settings.opencode].model must win, got %q", got["setting.model"])
	}
}

func TestDesiredSettings_NoRoutesNoModelKeys(t *testing.T) {
	got := desiredSettings(&config.Config{})
	if _, ok := got["setting.model"]; ok {
		t.Error("no model routes must not synthesize setting.model")
	}
	if _, ok := got["setting.small_model"]; ok {
		t.Error("no model routes must not synthesize setting.small_model")
	}
}
