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
	root.AddCommand(botCmd)
}

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "botを起動する",
	Long: `botの説明
	後で書く`,
	Run: func(cmd *cobra.Command, args []string) {
		bot.Run()
	},
}
