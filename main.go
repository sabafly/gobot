package main

import (
	"os"

	bot "github.com/sabafly/gobot/bot"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "gobot",
	Short: "とても便利でおいしいディスコードボット",
	// TODO: 書く
	Long: `後で書く`,
}

func main() {
	err := root.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	botCmd.Flags().StringP("config", "c", "config.json", "config file of bot")
	botCmd.Flags().String("gobot-config", "gobot.json", "config file of gobot")
	botCmd.Flags().StringP("lang", "l", "lang", "lang file path")
	root.AddCommand(botCmd)
}

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "botを起動する",
	Long: `botの説明
	後で書く`,
	ValidArgs: []string{
		"config",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return bot.Run(cmd.Flag("config").Value.String(), cmd.Flag("lang").Value.String(), cmd.Flag("gobot-config").Value.String())
	},
}
