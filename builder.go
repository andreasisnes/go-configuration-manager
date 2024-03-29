package configurationmanager

import (
	"github.com/andreasisnes/go-configuration-manager/modules"
)

type Builder interface {
	Add(module modules.Module) Builder
	Clear()
	Modules() []modules.Module
	Build() Configuration
}

type builder struct {
	modules []modules.Module
	options *Options
}

func New(options *Options) Builder {
	if options == nil {
		options = NewDefaultOptions()
	}

	return &builder{
		modules: []modules.Module{},
		options: options,
	}
}

func (b *builder) Add(module modules.Module) Builder {
	if module != nil {
		b.modules = append(b.modules, module)
	}

	return b
}

func (b *builder) Clear() {
	b.modules = []modules.Module{}
}

func (b *builder) Modules() []modules.Module {
	return b.modules
}

func (b *builder) Build() Configuration {
	return newConfiguration(b.options, b.Modules())
}
