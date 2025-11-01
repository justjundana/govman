package cli

import (
	"fmt"
	"os"

	viper "github.com/spf13/viper"
)

// init configures root-level persistent flags, binds them to viper,
// registers subcommands, and disables the default completion command.
// It runs automatically before main execution.
func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.govman/config.yaml)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
	rootCmd.PersistentFlags().Bool("quiet", false, "quiet output (errors only)")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to bind verbose flag: %v\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to bind quiet flag: %v\n", err)
		os.Exit(1)
	}

	addCommands()

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// addCommands registers all CLI subcommands to the root Cobra command.
func addCommands() {
	rootCmd.AddCommand(
		newInitCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newUseCmd(),
		newCurrentCmd(),
		newListCmd(),
		newInfoCmd(),
		newCleanCmd(),
		newSelfUpdateCmd(),
		newRefreshCmd(),
	)
}
