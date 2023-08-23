package commands

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/sabafly/disgo"
	"github.com/sabafly/disgo/discord"
	"github.com/sabafly/disgo/events"
	"github.com/sabafly/gobot/bot/client"
	lib "github.com/sabafly/sabafly-lib/v2"
	botlib "github.com/sabafly/sabafly-lib/v2/bot"
	"github.com/sabafly/sabafly-lib/v2/handler"
	"github.com/sabafly/sabafly-lib/v2/translate"
	"github.com/shirou/gopsutil/v3/mem"
)

func About(b *botlib.Bot[*client.Client]) handler.Command {
	return handler.Command{
		Create: discord.SlashCommandCreate{
			Name:                     "about",
			Description:              "show bot info",
			DescriptionLocalizations: translate.MessageMap("about_command_description", false),
			DMPermission:             &b.Config.DMPermission,
		},
		CommandHandlers: map[string]handler.CommandHandler{
			"": aboutCommandHandler(b),
		},
	}
}

func aboutCommandHandler(b *botlib.Bot[*client.Client]) handler.CommandHandler {
	return func(event *events.ApplicationCommandInteractionCreate) error {
		message := discord.NewMessageCreateBuilder()
		embed := discord.NewEmbedBuilder()
		embed.SetTitlef("%s Info", botlib.BotName)
		embed.SetDescriptionf("### **version**\r- %s\r### **%s**\r- %s\r - %s\r### **%s**\r- %s\r - %s\r### **go version**\r- %s",
			b.Version,
			lib.Name,
			lib.Module,
			lib.Version,
			disgo.Name,
			disgo.Module,
			disgo.Version,
			runtime.Version(),
		)
		gc_stat := new(debug.GCStats)
		debug.ReadGCStats(gc_stat)
		vm, err := mem.VirtualMemory()
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		if err != nil {
			embed.AddField("memory", fmt.Sprintf("Last GC:%v\r%s", discord.TimestampMention(gc_stat.LastGC.Unix()), err.Error()), true)
		} else {
			embed.AddField("memory",
				fmt.Sprintf(
					"Last GC:%v\rTotal: %dMB\rFree: %dMB\rUsed: %dMB\rUsed/Total: %.2f%%\rAlloc: %dMB\rTotalAlloc: %dMB\rHeapObjects: %d\rSys: %dMB",
					discord.TimestampMention(gc_stat.LastGC.Unix()),
					vm.Total/1024/1024,
					vm.Free/1024/1024,
					vm.Used/1024/1024,
					float64(vm.Used)/float64(vm.Total)*100,
					ms.Alloc/1024/1024,
					ms.TotalAlloc/1024/1024,
					ms.HeapObjects,
					ms.Sys/1024/1024,
				),
				false,
			)
		}
		embed.AddField("cpu",
			fmt.Sprintf(
				"NumCPU: %d\rNumGoroutine: %d\rGOMAXPROCS: %d",
				runtime.NumCPU(),
				runtime.NumGoroutine(),
				runtime.GOMAXPROCS(0),
			),
			false,
		)
		embed.AddField(
			"runtime",
			fmt.Sprintf(
				"GOOS: %s\rGOARCH: %s",
				runtime.GOOS,
				runtime.GOARCH,
			),
			false,
		)
		embed.Embed = botlib.SetEmbedProperties(embed.Embed)
		message.SetEmbeds(embed.Build())
		return event.CreateMessage(message.Build())
	}
}
