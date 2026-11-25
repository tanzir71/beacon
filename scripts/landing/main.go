// Command landing generates the three GitHub Pages entrypoints from one template.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const repoSlug = "tanzir71/beacon"

func main() {
	check := flag.Bool("check", false, "fail if generated pages are stale")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if err := generate(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(root string, check bool) error {
	owner, repository, _ := strings.Cut(repoSlug, "/")
	site := "https://" + owner + ".github.io/" + repository + "/"
	locales := make(map[string]map[string]string)
	for _, locale := range []string{"en", "bn"} {
		data, err := os.ReadFile(filepath.Join(root, "web/landing/i18n", locale+".json")) // #nosec G304 -- caller supplies repository root.
		if err != nil {
			return err
		}
		var text map[string]string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		locales[locale] = text
	}
	keys := func(values map[string]string) []string {
		result := make([]string, 0, len(values))
		for key := range values {
			result = append(result, key)
		}
		slices.Sort(result)
		return result
	}
	if !slices.Equal(keys(locales["en"]), keys(locales["bn"])) {
		return errors.New("landing keys differ between English and Banglish")
	}
	for _, output := range []struct{ path, locale, base string }{
		{"index.html", "en", "./"}, {"en/index.html", "en", "../"}, {"bn/index.html", "bn", "../"},
	} {
		text := locales[output.locale]
		page, err := template.New("landing.html").Funcs(template.FuncMap{
			"t": func(key string) (string, error) {
				value, found := text[key]
				if !found || strings.TrimSpace(value) == "" {
					return "", fmt.Errorf("missing %s landing key %s", output.locale, key)
				}
				return value, nil
			},
			"seq": func(count int) []int {
				values := make([]int, count)
				for index := range values {
					values[index] = index + 1
				}
				return values
			},
			"sub": func(value, amount int) int { return value - amount },
			"arc": func(level int) string {
				return []string{"M5 13.5a5.5 5.5 0 0 1 5.5 5.5", "M5 9.5A9.5 9.5 0 0 1 14.5 19", "M5 5.5A13.5 13.5 0 0 1 18.5 19", "M5 1.5A17.5 17.5 0 0 1 22.5 19"}[level-1]
			},
			"statusRows": func() []struct{ Key, State string } {
				return []struct{ Key, State string }{
					{"status.sites", "status.progress"}, {"status.interface", "status.progress"},
					{"status.messages", "status.planned"}, {"status.internet", "status.planned"},
					{"status.phones", "status.planned"}, {"status.offgrid", "status.planned"},
				}
			},
		}).ParseFiles(filepath.Join(root, "web/landing/landing.html"))
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		if err := page.Execute(&buffer, struct {
			Base, Locale, GitHub, Canonical string
			Root                            bool
		}{output.base, output.locale, "https://github.com/" + repoSlug,
			site + strings.TrimSuffix(output.path, "index.html"), output.path == "index.html"}); err != nil {
			return err
		}
		path := filepath.Join(root, output.path)
		if check {
			current, err := os.ReadFile(path) // #nosec G304 -- fixed generated output below repository root.
			if err != nil || !bytes.Equal(current, buffer.Bytes()) {
				return fmt.Errorf("%s is stale; run make landing", output.path)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
			return err
		}
	}
	return nil
}
