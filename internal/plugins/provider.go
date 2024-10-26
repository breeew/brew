package plugins

import (
	"github.com/starbx/brew-api/internal/core"
)

func Setup(install func(p core.Plugins), mode string) {
	p := provider[mode]
	if p == nil {
		panic("Setup mode not found: " + mode)
	}
	install(p())
}

var provider = map[string]core.SetupFunc{
	"selfhost": func() core.Plugins {
		return newSelfHostMode()
	},
	"saas": func() core.Plugins {
		return newSaaSPlugin()
	},
}
