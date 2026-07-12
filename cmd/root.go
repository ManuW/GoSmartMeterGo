package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"GoSmartMeterGo/pkg/config"
)

var (
	cfgFile string
	Cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "gosmartmeter",
	Short: "GoSmartMeterGo reads, aggregates and monitors energy metrics.",
	Long: `GoSmartMeterGo is a daemon that reads SML data from a Logarex LK13BE smart meter,
listens for multicast broadcasts from an SMA HomeManager 2.0, aggregates the data
into 5-minute averages, stores them in SQLite, and provides a web dashboard and Telegram alerts.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		Cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default is ./config.yaml)")
}
