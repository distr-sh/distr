package cmd

import (
	"github.com/spf13/cobra"
)

func NewLicenseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "license-key",
	}
	cmd.AddCommand(
		NewGenerateLicenseKeyCommand(),
	)
	return cmd
}

func init() {
	RootCommand.AddCommand(NewLicenseCommand())
}
