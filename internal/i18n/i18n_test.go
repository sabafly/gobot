package i18n

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template/parse"

	"github.com/disgoorg/disgo/discord"
)

var legacyPlaceholderPattern = regexp.MustCompile(`\{[a-zA-Z0-9_]+\}`)

func extractTemplateFields(tmplStr string) (map[string]struct{}, error) {
	tmpl := newTemplate("check")
	parsed, err := tmpl.Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]struct{})
	if parsed.Tree != nil && parsed.Root != nil {
		walkParseNode(parsed.Root, fields)
	}
	return fields, nil
}

func walkParseNode(node parse.Node, fields map[string]struct{}) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		if n != nil {
			for _, child := range n.Nodes {
				walkParseNode(child, fields)
			}
		}
	case *parse.ActionNode:
		walkParseNode(n.Pipe, fields)
	case *parse.PipeNode:
		if n != nil {
			for _, cmd := range n.Cmds {
				walkParseNode(cmd, fields)
			}
		}
	case *parse.CommandNode:
		if n != nil {
			for _, arg := range n.Args {
				walkParseNode(arg, fields)
			}
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			fields[n.Ident[0]] = struct{}{}
		}
	case *parse.IfNode:
		walkParseNode(n.Pipe, fields)
		walkParseNode(n.List, fields)
		walkParseNode(n.ElseList, fields)
	case *parse.RangeNode:
		walkParseNode(n.Pipe, fields)
		walkParseNode(n.List, fields)
		walkParseNode(n.ElseList, fields)
	case *parse.WithNode:
		walkParseNode(n.Pipe, fields)
		walkParseNode(n.List, fields)
		walkParseNode(n.ElseList, fields)
	}
}

func collectComponentTexts(comp Component) []string {
	var texts []string
	switch c := comp.(type) {
	case TextDisplay:
		texts = append(texts, string(c))
	case Button:
		texts = append(texts, c.Label)
	case StringSelectMenu:
		texts = append(texts, c.Placeholder)
		for _, opt := range c.Options.opts {
			texts = append(texts, opt.Label, opt.Description)
		}
	case UserSelectMenu:
		texts = append(texts, c.Placeholder)
	case RoleSelectMenu:
		texts = append(texts, c.Placeholder)
	case MentionableSelectMenu:
		texts = append(texts, c.Placeholder)
	case ChannelSelectMenu:
		texts = append(texts, c.Placeholder)
	case ActionRow:
		for _, sub := range c.Components {
			texts = append(texts, collectComponentTexts(sub)...)
		}
	case Section:
		for _, sub := range c.Components {
			texts = append(texts, collectComponentTexts(sub)...)
		}
		if c.Accessory != nil {
			texts = append(texts, collectComponentTexts(c.Accessory)...)
		}
	case Thumbnail:
		texts = append(texts, c.Description)
	case MediaGallery:
		for _, item := range c.Media {
			texts = append(texts, item.Description)
		}
	case Container:
		for _, sub := range c.Components {
			texts = append(texts, collectComponentTexts(sub)...)
		}
	case TextInput:
		texts = append(texts, c.Placeholder, c.Value)
	case Label:
		texts = append(texts, c.Label, c.Description)
		if c.Component != nil {
			texts = append(texts, collectComponentTexts(c.Component)...)
		}
	}
	return texts
}

func TestLoadLocalesAndSyntax(t *testing.T) {
	localesDir := filepath.Join("..", "..", "locales")
	if err := LoadLocales(localesDir); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	if len(globalLocales) == 0 {
		t.Fatal("expected at least one locale loaded")
	}
	defValues, ok := globalLocales[defaultLocale]
	if !ok {
		t.Fatalf("default locale %q not found in globalLocales", defaultLocale)
	}

	allowedLegacyPlaceholderKeys := map[string]bool{
		// User-facing help text showing literal {user}, {username}, etc. placeholders for custom level-up messages
		"components.level.up.message.modal.input.message.placeholder": true,
	}

	for locale, values := range globalLocales {
		for key, str := range values.Strings {
			fields, err := extractTemplateFields(str)
			if err != nil {
				t.Errorf("locale %s string key %q has invalid template syntax: %v", locale, key, err)
				continue
			}

			// Check for unmigrated {var} placeholders
			cleaned := regexp.MustCompile(`\{\{.*?\}\}`).ReplaceAllString(str, "")
			if !allowedLegacyPlaceholderKeys[key] && legacyPlaceholderPattern.MatchString(cleaned) {
				t.Errorf("locale %s string key %q still contains legacy {var} placeholder: %s", locale, key, str)
			}

			// Check that template variables match default locale
			if locale != defaultLocale {
				if defStr, exists := defValues.Strings[key]; exists {
					defFields, err := extractTemplateFields(defStr)
					if err == nil && !maps.Equal(fields, defFields) {
						t.Errorf("locale %s string key %q template fields %v do not match default locale fields %v", locale, key, slices.Collect(maps.Keys(fields)), slices.Collect(maps.Keys(defFields)))
					}
				}
			}
		}

		for key, comps := range values.Components {
			for i, comp := range comps {
				for _, txt := range collectComponentTexts(comp) {
					if _, err := extractTemplateFields(txt); err != nil {
						t.Errorf("locale %s component key %q [%d] has invalid template syntax: %v", locale, key, i, err)
					}
					cleaned := regexp.MustCompile(`\{\{.*?\}\}`).ReplaceAllString(txt, "")
					if legacyPlaceholderPattern.MatchString(cleaned) {
						t.Errorf("locale %s component key %q [%d] still contains legacy {var} placeholder in text: %s", locale, key, i, txt)
					}
				}
			}
		}
	}
}

func TestAllCallSitesStaticAnalysis(t *testing.T) {
	localesDir := filepath.Join("..", "..", "locales")
	if err := LoadLocales(localesDir); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}
	defValues := globalLocales[defaultLocale]

	rootDir := filepath.Join("..", "..")
	fset := token.NewFileSet()

	type fileAST struct {
		path string
		file *ast.File
	}
	var files []fileAST

	// Collect package-level string constants and helper functions across the codebase
	constStrings := make(map[string]string)
	funcMapHelpers := make(map[string]map[string]struct{})

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}
		files = append(files, fileAST{path: path, file: f})

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.CONST {
					continue
				}
				for _, spec := range d.Specs {
					vSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vSpec.Names {
						if i < len(vSpec.Values) {
							if lit, ok := vSpec.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								if val, err := strconv.Unquote(lit.Value); err == nil {
									constStrings[name.Name] = val
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				ast.Inspect(d.Body, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok {
						return true
					}
					for _, lhs := range assign.Lhs {
						if idx, ok := lhs.(*ast.IndexExpr); ok {
							if lit, ok := idx.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING {
								if val, err := strconv.Unquote(lit.Value); err == nil {
									if funcMapHelpers[d.Name.Name] == nil {
										funcMapHelpers[d.Name.Name] = make(map[string]struct{})
									}
									funcMapHelpers[d.Name.Name][val] = struct{}{}
								}
							}
						}
					}
					return true
				})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir failed: %v", err)
	}

	resolveString := func(expr ast.Expr, localVars map[string][]string) ([]string, bool) {
		switch e := expr.(type) {
		case *ast.BasicLit:
			if e.Kind == token.STRING {
				val, err := strconv.Unquote(e.Value)
				if err == nil {
					return []string{val}, true
				}
			}
		case *ast.Ident:
			if vals, ok := localVars[e.Name]; ok && len(vals) > 0 {
				return vals, true
			}
			if val, ok := constStrings[e.Name]; ok {
				return []string{val}, true
			}
		}
		return nil, false
	}

	extractMapKeys := func(expr ast.Expr, localMapVars map[string]map[string]struct{}) map[string]struct{} {
		keys := make(map[string]struct{})
		var visit func(e ast.Expr)
		visit = func(e ast.Expr) {
			if e == nil {
				return
			}
			switch v := e.(type) {
			case *ast.CompositeLit:
				for _, elt := range v.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if k, ok := resolveString(kv.Key, nil); ok {
							for _, keyStr := range k {
								keys[keyStr] = struct{}{}
							}
						}
					}
				}
			case *ast.Ident:
				if mKeys, ok := localMapVars[v.Name]; ok {
					for mk := range mKeys {
						keys[mk] = struct{}{}
					}
				}
			case *ast.CallExpr:
				if ident, ok := v.Fun.(*ast.Ident); ok {
					if helperKeys, ok := funcMapHelpers[ident.Name]; ok {
						for hk := range helperKeys {
							keys[hk] = struct{}{}
						}
					}
				}
				for _, arg := range v.Args {
					visit(arg)
				}
			}
		}
		visit(expr)
		return keys
	}

	for _, fa := range files {
		// Collect all .WithText("key", ...) keys and local variables in each function body
		var inspectFuncBody = func(fnName string, body *ast.BlockStmt) {
			localStringVars := make(map[string][]string)
			localMapVars := make(map[string]map[string]struct{})
			withTextKeysInFunc := make(map[string]struct{})

			ast.Inspect(body, func(n ast.Node) bool {
				switch stmt := n.(type) {
				case *ast.AssignStmt:
					for i, lhs := range stmt.Lhs {
						switch l := lhs.(type) {
						case *ast.Ident:
							if i < len(stmt.Rhs) {
								switch r := stmt.Rhs[i].(type) {
								case *ast.BasicLit:
									if r.Kind == token.STRING {
										if val, err := strconv.Unquote(r.Value); err == nil {
											localStringVars[l.Name] = append(localStringVars[l.Name], val)
										}
									}
								case *ast.CompositeLit:
									for _, elt := range r.Elts {
										if kv, ok := elt.(*ast.KeyValueExpr); ok {
											if ks, ok := resolveString(kv.Key, nil); ok {
												if localMapVars[l.Name] == nil {
													localMapVars[l.Name] = make(map[string]struct{})
												}
												for _, k := range ks {
													localMapVars[l.Name][k] = struct{}{}
												}
											}
										}
									}
								}
							}
						case *ast.IndexExpr:
							if mapIdent, ok := l.X.(*ast.Ident); ok {
								if ks, ok := resolveString(l.Index, nil); ok {
									if localMapVars[mapIdent.Name] == nil {
										localMapVars[mapIdent.Name] = make(map[string]struct{})
									}
									for _, k := range ks {
										localMapVars[mapIdent.Name][k] = struct{}{}
									}
								}
							}
						}
					}
				case *ast.ValueSpec:
					for i, ident := range stmt.Names {
						if i < len(stmt.Values) {
							if lit, ok := stmt.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								if val, err := strconv.Unquote(lit.Value); err == nil {
									localStringVars[ident.Name] = append(localStringVars[ident.Name], val)
								}
							}
						}
					}
				case *ast.CallExpr:
					if sel, ok := stmt.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "WithText" && len(stmt.Args) >= 1 {
						if ks, ok := resolveString(stmt.Args[0], localStringVars); ok {
							for _, k := range ks {
								withTextKeysInFunc[k] = struct{}{}
							}
						}
					}
				}
				return true
			})

			ast.Inspect(body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				pos := fset.Position(call.Pos())
				loc := fmt.Sprintf("%s:%d", fa.path, pos.Line)

				switch pkgIdent.Name {
				case "i18n":
					switch sel.Sel.Name {
					case "TranslateText":
						if len(call.Args) < 2 {
							return true
						}

						if strings.HasSuffix(fa.path, filepath.Join("internal", "i18n", "translates.go")) ||
							strings.HasSuffix(fa.path, filepath.Join("internal", "errors", "errors.go")) ||
							strings.HasSuffix(fa.path, filepath.Join("bot", "commands", "debug", "debug.go")) {
							return true
						}
						keys, ok := resolveString(call.Args[1], localStringVars)
						if !ok {
							t.Errorf("%s: could not statically resolve key in i18n.TranslateText call", loc)
							return true
						}
						providedKeys := make(map[string]struct{})
						if len(call.Args) >= 3 {
							providedKeys = extractMapKeys(call.Args[2], localMapVars)
						}
						for _, key := range keys {
							tmplStr, exists := defValues.Strings[key]
							if !exists {
								t.Errorf("%s: i18n.TranslateText key %q does not exist in default.yml strings", loc, key)
								continue
							}
							reqFields, err := extractTemplateFields(tmplStr)
							if err != nil {
								t.Errorf("%s: key %q has invalid template: %v", loc, key, err)
								continue
							}
							for req := range reqFields {
								if _, ok := providedKeys[req]; !ok {
									t.Errorf("%s: i18n.TranslateText(%q) is missing required template argument %q (provided: %v)", loc, key, req, slices.Collect(maps.Keys(providedKeys)))
								}
							}
						}

					case "TranslateTextMap", "TranslateCommandOptionMap":
						if len(call.Args) < 1 {
							return true
						}
						if strings.HasSuffix(fa.path, filepath.Join("internal", "i18n", "translates.go")) {
							return true
						}
						keys, ok := resolveString(call.Args[0], localStringVars)
						if !ok {
							t.Errorf("%s: could not statically resolve key in i18n.%s call", loc, sel.Sel.Name)
							return true
						}
						for _, key := range keys {
							if _, exists := defValues.Strings[key]; !exists {
								t.Errorf("%s: i18n.%s key %q does not exist in default.yml strings", loc, sel.Sel.Name, key)
							}
						}

					case "TranslateComponent", "TranslateLayout":
						if len(call.Args) < 2 {
							return true
						}
						if strings.HasSuffix(fa.path, filepath.Join("internal", "i18n", "translates.go")) {
							return true
						}
						keys, ok := resolveString(call.Args[1], localStringVars)
						if !ok {
							t.Errorf("%s: could not statically resolve key in i18n.%s call", loc, sel.Sel.Name)
							return true
						}
						for _, key := range keys {
							comps, exists := defValues.Components[key]
							if !exists {
								t.Errorf("%s: i18n.%s key %q does not exist in default.yml components", loc, sel.Sel.Name, key)
								continue
							}
							for _, comp := range comps {
								for _, txt := range collectComponentTexts(comp) {
									reqFields, err := extractTemplateFields(txt)
									if err != nil {
										continue
									}
									for req := range reqFields {
										if _, ok := withTextKeysInFunc[req]; !ok {
											t.Errorf("%s: i18n.%s(%q) requires WithText(%q, ...) in function %s, but only found %v", loc, sel.Sel.Name, key, req, fnName, slices.Collect(maps.Keys(withTextKeysInFunc)))
										}
									}
								}
							}
						}
					}

				case "errors":
					if sel.Sel.Name == "ErrorMessage" && len(call.Args) >= 1 {
						keys, ok := resolveString(call.Args[0], localStringVars)
						if !ok {
							t.Errorf("%s: could not statically resolve key in errors.ErrorMessage call", loc)
							return true
						}
						providedKeys := make(map[string]struct{})
						hasCustomDesc := false
						for _, arg := range call.Args[2:] {
							if cExpr, ok := arg.(*ast.CallExpr); ok {
								if sExpr, ok := cExpr.Fun.(*ast.SelectorExpr); ok && sExpr.Sel.Name == "WithDescription" {
									hasCustomDesc = true
								}
							}
							for k := range extractMapKeys(arg, localMapVars) {
								providedKeys[k] = struct{}{}
							}
						}
						for _, key := range keys {
							checkKeys := []string{key}
							if !hasCustomDesc {
								checkKeys = append(checkKeys, key+".description")
							}
							for _, checkKey := range checkKeys {
								tmplStr, exists := defValues.Strings[checkKey]
								if !exists {
									t.Errorf("%s: errors.ErrorMessage key %q does not exist in default.yml strings", loc, checkKey)
									continue
								}
								reqFields, err := extractTemplateFields(tmplStr)
								if err != nil {
									t.Errorf("%s: key %q has invalid template: %v", loc, checkKey, err)
									continue
								}
								for req := range reqFields {
									if _, ok := providedKeys[req]; !ok {
										t.Errorf("%s: errors.ErrorMessage(%q) is missing required template argument %q for %q", loc, key, req, checkKey)
									}
								}
							}
						}
					}
				}
				return true
			})
		}

		for _, decl := range fa.file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil {
				inspectFuncBody(fn.Name.Name, fn.Body)
			}
		}

		// Also inspect struct/slice composite literals at package level (e.g., CommandCreate definitions outside functions)
		for _, decl := range fa.file.Decls {
			if _, ok := decl.(*ast.FuncDecl); ok {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "i18n" {
					return true
				}
				pos := fset.Position(call.Pos())
				loc := fmt.Sprintf("%s:%d", fa.path, pos.Line)
				if (sel.Sel.Name == "TranslateTextMap" || sel.Sel.Name == "TranslateCommandOptionMap") && len(call.Args) >= 1 {
					keys, ok := resolveString(call.Args[0], nil)
					if !ok {
						t.Errorf("%s: could not statically resolve key in i18n.%s call", loc, sel.Sel.Name)
						return true
					}
					for _, key := range keys {
						if _, exists := defValues.Strings[key]; !exists {
							t.Errorf("%s: i18n.%s key %q does not exist in default.yml strings", loc, sel.Sel.Name, key)
						}
					}
				}
				return true
			})
		}
	}
}

func TestSecurity_NoSSTIOrDoubleExpansion(t *testing.T) {
	localesDir := filepath.Join("..", "..", "locales")
	if err := LoadLocales(localesDir); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	t.Run("TranslateText_UserInputContainingTemplateSyntaxIsNotExecuted", func(t *testing.T) {
		maliciousSuffix := "{{.user}} {{printf \"%s\" \"injected\"}}"
		got := TranslateText(discord.LocaleJapanese, "components.message.suffix.set.message", map[string]any{
			"user":   "@alice",
			"suffix": maliciousSuffix,
		})
		if !strings.Contains(got, maliciousSuffix) {
			t.Fatalf("expected literal malicious suffix %q in output, got: %q", maliciousSuffix, got)
		}
		if strings.Contains(got, "injected") && !strings.Contains(got, "{{printf") {
			t.Fatalf("SSTI detected: template directives in user input were evaluated: %q", got)
		}
	})

	t.Run("MapContext_replaceText_SinglePassNoSecondOrderExpansion", func(t *testing.T) {
		ctx := BuildContext().
			WithText("a", "{{.b}}").
			WithText("b", "SECRET_VALUE")

		out := ctx.replaceText("value is {{.a}}")
		if out != "value is {{.b}}" {
			t.Fatalf("expected single-pass output %q without expanding {{.b}}, got %q", "value is {{.b}}", out)
		}
	})

	t.Run("MapContext_ReplaceCustomID_SinglePassNoSecondOrderExpansion", func(t *testing.T) {
		ctx := BuildContext().
			WithCustomID("a", "{b}").
			WithCustomID("b", "INJECTED_ID")

		out := ctx.ReplaceCustomID("action:{a}:{b}")
		if out != "action:{b}:INJECTED_ID" {
			t.Fatalf("expected single-pass CustomID replacement %q, got %q", "action:{b}:INJECTED_ID", out)
		}
	})
}

func TestLocaleHelpers(t *testing.T) {
	localesDir := filepath.Join("..", "..", "locales")
	if err := LoadLocales(localesDir); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	if got := DefaultLocale(); got != discord.LocaleJapanese {
		t.Fatalf("expected DefaultLocale ja, got %q", got)
	}
	locs := AvailableLocales()
	if len(locs) < 2 {
		t.Fatalf("expected at least 2 locales, got %v", locs)
	}
	if locs[0] != discord.LocaleJapanese {
		t.Fatalf("expected default locale ja first, got %v", locs)
	}
	if !slices.Contains(locs, discord.LocaleEnglishUS) {
		t.Fatalf("expected AvailableLocales to contain en-US, got %v", locs)
	}
	locs2 := AvailableLocales()
	if !slices.Equal(locs, locs2) {
		t.Fatalf("expected deterministic AvailableLocales order: %v vs %v", locs, locs2)
	}

	if !IsSupportedLocale(discord.LocaleJapanese) || !IsSupportedLocale(discord.LocaleEnglishUS) {
		t.Fatal("expected ja and en-US to be supported")
	}
	if IsSupportedLocale("") || IsSupportedLocale(discord.LocaleUnknown) || IsSupportedLocale(discord.Locale("xx-INVALID")) {
		t.Fatal("expected empty/unknown/invalid locale not to be supported")
	}

	if got := LocaleDisplayName(discord.LocaleJapanese); got != "日本語 (ja)" {
		t.Fatalf("unexpected display name for ja: %q", got)
	}
	if got := LocaleDisplayName(discord.LocaleEnglishUS); got != "English (US) (en-US)" {
		t.Fatalf("unexpected display name for en-US: %q", got)
	}
	if got := LocaleDisplayName(""); got != "日本語 (ja)" {
		t.Fatalf("unexpected display name for empty locale: %q", got)
	}
	if got := LocaleDisplayName(discord.LocaleKorean); got != "Korean (ko)" {
		t.Fatalf("unexpected display name for ko: %q", got)
	}
}

func TestLocaleComponentTemplatesParity(t *testing.T) {
	localesDir := filepath.Join("..", "..", "locales")
	if err := LoadLocales(localesDir); err != nil {
		t.Fatalf("LoadLocales failed: %v", err)
	}

	defValues, ok := globalLocales[defaultLocale]
	if !ok {
		t.Fatalf("default locale %q not found", defaultLocale)
	}
	enValues, ok := globalLocales[discord.LocaleEnglishUS]
	if !ok {
		t.Fatalf("en-US locale not found")
	}

	collectFieldsForKey := func(loc discord.Locale, key string, comps []Component) map[string]struct{} {
		fields := make(map[string]struct{})
		for i, c := range comps {
			for _, txt := range collectComponentTexts(c) {
				f, err := extractTemplateFields(txt)
				if err != nil {
					t.Errorf("locale %s component %q [%d] parse failed: %v", loc, key, i, err)
					continue
				}
				for k := range f {
					fields[k] = struct{}{}
				}
			}
		}
		return fields
	}

	for key, defComps := range defValues.Components {
		enComps, exists := enValues.Components[key]
		if !exists {
			t.Errorf("en-US is missing components template key %q", key)
			continue
		}
		if len(enComps) != len(defComps) {
			t.Errorf("en-US components[%q] length=%d, want %d", key, len(enComps), len(defComps))
		}
		defFields := collectFieldsForKey(defaultLocale, key, defComps)
		enFields := collectFieldsForKey(discord.LocaleEnglishUS, key, enComps)
		if !maps.Equal(defFields, enFields) {
			t.Errorf("en-US components[%q] template fields %v do not match default fields %v",
				key, slices.Collect(maps.Keys(enFields)), slices.Collect(maps.Keys(defFields)))
		}
	}

	layout := BuildContext().
		WithText("guild_name", "TestGuild").
		WithText("guild_locale", "English (US) (en-US) (Overridden)").
		WithText("bump_enabled", "Enabled").
		WithText("bump_message_title", "Bump Title").
		WithText("bump_message", "Bump Body").
		WithText("bump_remind_message_title", "Remind Title").
		WithText("bump_remind_message", "Remind Body").
		WithText("bump_mention", "None").
		WithText("up_enabled", "Enabled").
		WithText("up_message_title", "Up Title").
		WithText("up_message", "Up Body").
		WithText("up_remind_message_title", "Up Remind").
		WithText("up_remind_message", "Up Remind Body").
		WithText("up_mention", "None").
		WithText("leveling_enabled", "Disabled").
		Translate(TranslateLayout(discord.LocaleEnglishUS, "command.settings.view"))
	if len(layout) != 1 {
		t.Fatalf("expected 1 container layout, got %d", len(layout))
	}
	cont, ok := layout[0].(discord.ContainerComponent)
	if !ok {
		t.Fatalf("expected ContainerComponent, got %T", layout[0])
	}
	var joined strings.Builder
	for _, sub := range cont.Components {
		if td, ok := sub.(discord.TextDisplayComponent); ok {
			joined.WriteString(td.Content)
			joined.WriteByte('\n')
		}
	}
	rendered := joined.String()
	if !strings.Contains(rendered, "TestGuild Settings") || !strings.Contains(rendered, "**Guild Language**: English (US) (en-US) (Overridden)") {
		t.Fatalf("unexpected rendered en-US settings view:\n%s", rendered)
	}
}
