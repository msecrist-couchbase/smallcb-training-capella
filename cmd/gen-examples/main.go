package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

var codeMaxLen = 8000

var notYetSupported = []string{
	"nalytics", // For Analytics/analytics.
	"rxjava",
	"searchQuery",
	"select ...",
	"ssl",
	"viewQuery",
}

// Sibling directories to scan for examples.
var exampleTypeDirs = [][]string{
	[]string{"go", "../sdk-examples/go"},

	[]string{"java", "../docs-sdk-java/modules/howtos/examples"},

	[]string{"js", "../docs-sdk-nodejs/modules/hello-world/examples/getting-started"},
	[]string{"js", "../docs-sdk-nodejs/modules/devguide/examples/nodejs"},
	[]string{"js", "../docs-sdk-nodejs/modules/howtos/examples"},

	[]string{"py", "../docs-sdk-python/modules/hello-world/examples"},
	[]string{"py", "../docs-sdk-python/modules/devguide/examples/python"},
	[]string{"py", "../docs-sdk-python/modules/howtos/examples"},
}

// Mapping from a file suffix to longer names.
var suffixToName = map[string]string{
	"go":   "go",
	"java": "java",
	"js":   "nodejs",
	"py":   "python",
}

// Mapping from a file suffix to lang.
var suffixToLang = map[string]string{
	"go":   "go",
	"java": "java",
	"js":   "nodejs",
	"py":   "py",
}

func main() {
	var err error

	m := map[string]map[string]string{}

	for _, exampleTypeDir := range exampleTypeDirs {
		m, err = ReadFiles(exampleTypeDir[1],
			map[string]bool{
				exampleTypeDir[0]: true,
			}, m)
		if err != nil {
			log.Fatal(err)
		}
	}

	for suffix, m2 := range m {
		for name, code := range m2 {
			log.Printf("suffix: %s, name: %s, ...", suffix, name)

			code, rejectReason := CodeCleanse(suffix, code)
			if rejectReason != "" {
				log.Printf("suffix: %s, name: %s, ...SKIPPED: %s",
					suffix, name, rejectReason)
				continue
			}

			d := map[string]string{
				"chapter": suffixToName[suffix],
				"page":    "",
				"title":   suffix + ": " + strings.ReplaceAll(name, "-", " "),
				"lang":    suffixToLang[suffix],
				"code":    code,
			}

			b, err := yaml.Marshal(d)
			if err != nil {
				log.Fatal(err)
			}

			outName := fmt.Sprintf(
				"./cmd/play-server/static/examples/gen_%s_%s.yaml",
				suffix, name)

			err = ioutil.WriteFile(outName, b, 0644)
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("suffix: %s, name: %s, ...OK: %s",
				suffix, name, outName)
		}
	}
}

func CodeCleanse(suffix, code string) (codeNew, rejectReason string) {
	if len(code) > codeMaxLen {
		return "", fmt.Sprintf("code too long, %d > %d",
			len(code), codeMaxLen)
	}

	for _, s := range notYetSupported {
		if strings.Index(code, s) >= 0 {
			return "", "not yet supported: " + s
		}
	}

	if strings.Index(code, "beer-sample") < 0 &&
		strings.Index(code, "travel-sample") < 0 {
		return "", "no bs/ts bucket"
	}

	codeNew = strings.ReplaceAll(code, "'Administrator'", "'{{.CBUser}}'")
	codeNew = strings.ReplaceAll(codeNew, "\"Administrator\"", "\"{{.CBUser}}\"")
	if codeNew == code {
		return "", "no user"
	}
	code = codeNew

	codeNew = strings.ReplaceAll(code, "'password'", "'{{.CBPswd}}'")
	codeNew = strings.ReplaceAll(codeNew, "\"password\"", "\"{{.CBPswd}}\"")
	if codeNew == code {
		return "", "no pswd"
	}
	code = codeNew

	codeNew = strings.ReplaceAll(code, "127.0.0.1", "{{.Host}}")
	codeNew = strings.ReplaceAll(codeNew, "localhost", "{{.Host}}")
	if codeNew == code {
		return "", "no host"
	}
	code = codeNew

	if suffix == "java" {
		code = rePublicClass.ReplaceAllString(code, "class Program {")
	}

	for _, reTag := range reTags {
		code = reTag.ReplaceAllString(code, "")
	}

	code = reNewlines.ReplaceAllString(code, "\n\n")

	return code, ""
}

var reNewlines = regexp.MustCompile(`\n\n\n+`)

var rePublicClass = regexp.MustCompile(`(public )?class ([A-Z][a-zA-Z]+) {`)

var reTags = []*regexp.Regexp{
	regexp.MustCompile(`\/\/ #?tag::([a-z\-]+)\[\]\n`),
	regexp.MustCompile(`\/\/ #?end::([a-z\-]+)\[\]\n`),

	regexp.MustCompile(`#\s?tag::([a-z\-]+)\[\]\n`),
	regexp.MustCompile(`#\s?end::([a-z\-]+)\[\]\n`),

	regexp.MustCompile(`"""\n\[source,([a-z]+)\]\n----\n"""\n`),
}

func ReadFiles(dir string, suffixes map[string]bool,
	// Keyed by suffix (i.e., "go"), then by basename, and value is contents.
	rv map[string]map[string]string) (
	map[string]map[string]string, error) {
	if rv == nil {
		rv = map[string]map[string]string{}
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadDir, dir: %s, err: %v", dir, err)
	}

	for _, f := range files {
		name := f.Name()

		parts := strings.Split(name, ".")

		suffix := parts[len(parts)-1]

		if suffixes[suffix] {
			path := dir + "/" + name

			b, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("ioutil.ReadFile, path: %s, err: %v", path, err)
			}

			m2 := rv[suffix]
			if m2 == nil {
				m2 = map[string]string{}
				rv[suffix] = m2
			}

			m2[name[0:len(name)-len(suffix)-1]] = string(b)
		}
	}

	return rv, nil
}
