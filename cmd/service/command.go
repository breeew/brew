package service

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/starbx/brew-api/internal/core"
	"github.com/starbx/brew-api/internal/logic/v1/process"
	"github.com/starbx/brew-api/internal/plugins"
)

type Options struct {
	ConfigPath string
	Init       string
}

func (o *Options) AddFlags(flagSet *pflag.FlagSet) {
	// Add flags for generic options
	flagSet.StringVarP(&o.ConfigPath, "config", "c", "", "init api by given config")
	flagSet.StringVarP(&o.Init, "init", "i", "selfhost", "start service after initialize")
}

func NewCommand() *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "service",
		Short: "chat service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(opts)
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

func Run(opts *Options) error {

	app := core.MustSetupCore(core.MustLoadBaseConfig(opts.ConfigPath))
	plugins.Setup(app.InstallPlugins, opts.Init)
	process.StartKnowledgeProcess(app, 10)
	serve(app)

	return nil
}
