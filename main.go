package main

import (
	"fmt"
	"os"

	bot "github.com/sabafly/gobot/bot"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "gobot",
	Short: "とても便利でおいしいディスコードボット",
}

func main() {
	err := root.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	root.AddCommand(bot.Command())
}
