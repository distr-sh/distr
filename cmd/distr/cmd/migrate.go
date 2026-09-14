package cmd

import (
	"context"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/migrations"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/util"
	"github.com/spf13/cobra"
)

type MigrateOptions struct {
	Down bool
	To   uint
}

var migrateOpts = MigrateOptions{}

var MigrateCommand = &cobra.Command{
	Use:    "migrate",
	Short:  "execute database migrations",
	Args:   cobra.NoArgs,
	PreRun: func(cmd *cobra.Command, args []string) { env.Initialize() },
	Run: func(cmd *cobra.Command, args []string) {
		runMigrate(cmd.Context(), migrateOpts)
	},
}

func init() {
	MigrateCommand.Flags().BoolVar(&migrateOpts.Down, "down", migrateOpts.Down,
		"run all down migrations. DANGER: This will purge the database!")
	MigrateCommand.Flags().UintVar(&migrateOpts.To, "to", migrateOpts.To,
		"run all up/down migrations to reach specified schema revision")
	MigrateCommand.MarkFlagsMutuallyExclusive("down", "to")

	RootCommand.AddCommand(MigrateCommand)
}

func runMigrate(ctx context.Context, opts MigrateOptions) {
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { util.Must(registry.Shutdown(ctx)) }()
	if opts.To > 0 {
		registry.GetLogger().Sugar().Infof("run migrations to schema version %v", opts.To)
		util.Must(migrations.Migrate(registry.GetLogger(), opts.To))
	} else if opts.Down {
		registry.GetLogger().Info("run DOWN migrations")
		util.Must(migrations.Down(registry.GetLogger()))
	} else {
		registry.GetLogger().Info("run UP migrations")
		util.Must(migrations.Up(registry.GetLogger()))
	}
}
