package main

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var eventHandlerTmpl = template.Must(template.New("eventHandler").Funcs(template.FuncMap{
	"constName":      constName,
	"isDiscordEvent": isDiscordEvent,
	"privateName":    privateName,
}).Parse(`// "eventhandlers.go"から生成されています; 編集禁止
// events.go を確認

package gobot

// Following are all the event types.
// Event type values are used to match the events returned by Discord.
// EventTypes surrounded by __ are synthetic and are internal to DiscordGo.
const ({{range .}}
  {{privateName .}}EventType = "{{constName .}}"{{end}}
)
{{range .}}
// {{.}} イベントのイベントハンダラを返します
type {{privateName .}}EventHandler func(*Shard, *{{.}})

// {{.}} イベントの型名を返します
func (eh {{privateName .}}EventHandler) Type() string {
  return {{privateName .}}EventType
}
{{if isDiscordEvent .}}
// {{.}} の新しいインスタンスを返します
func (eh {{privateName .}}EventHandler) New() any {
  return &{{.}}{}
}{{end}}
// {{.}} イベントのハンダラ
func (eh {{privateName .}}EventHandler) Handle(s *Shard, i any) {
  if t, ok := i.(*{{.}}); ok {
    eh(s, t)
  }
}

{{end}}
func handlerForInterface(handler any) EventHandler {
  switch v := handler.(type) {
  case func(*Shard, any):
    return anyEventHandler(v){{range .}}
  case func(*Shard, *{{.}}):
    return {{privateName .}}EventHandler(v){{end}}
  }

  return nil
}

func init() { {{range .}}{{if isDiscordEvent .}}
  registerInterfaceProvider({{privateName .}}EventHandler(nil)){{end}}{{end}}
}
`))

func main() {
	var buf bytes.Buffer
	dir := filepath.Dir("./pkg/bot/")

	fs := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fs, "pkg/bot/events.go", nil, 0)
	if err != nil {
		log.Fatalf("warning: internal error: could not parse events.go: %s", err)
		return
	}

	names := []string{}
	for object := range parsedFile.Scope.Objects {
		names = append(names, object)
	}
	sort.Strings(names)
	eventHandlerTmpl.Execute(&buf, names)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Println("warning: internal error: invalid Go generated:", err)
		src = buf.Bytes()
	}

	err = os.WriteFile(filepath.Join(dir, strings.ToLower("eventhandlers.go")), src, 0644)
	if err != nil {
		log.Fatal(buf, "writing output: %s", err)
	}
}

var constRegexp = regexp.MustCompile("([a-z])([A-Z])")

func constCase(name string) string {
	return strings.ToUpper(constRegexp.ReplaceAllString(name, "${1}_${2}"))
}

func isDiscordEvent(name string) bool {
	switch {
	case name == "Connect", name == "Disconnect", name == "Event", name == "RateLimit", name == "Interface":
		return false
	default:
		return true
	}
}

func constName(name string) string {
	if !isDiscordEvent(name) {
		return "__" + constCase(name) + "__"
	}

	return constCase(name)
}

func privateName(name string) string {
	return strings.ToLower(string(name[0])) + name[1:]
}
