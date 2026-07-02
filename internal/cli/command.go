package cli

// init configures root-level persistent flags,
// registers subcommands, and disables the default completion command.
// It runs automatically before main execution.
func init() {
	rootCmd.SetFlagErrorFunc(flagUsageError)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.govman/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "quiet output (errors only)")

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
		newPruneCmd(),
		newSelfUpdateCmd(),
		newRefreshCmd(),
	)
}
