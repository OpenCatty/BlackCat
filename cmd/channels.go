package cmd

import "github.com/spf13/cobra"

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Manage messaging channel connections",
	Long:  `Configure and manage WhatsApp, Telegram, and Discord channel connections.`,
}

func init() {
	rootCmd.AddCommand(channelsCmd)
}
