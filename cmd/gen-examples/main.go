package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/yaml.v2"
)

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

var suffixToLang = map[string]string{
	"go":   "go",
	"java": "java",
	"js":   "nodejs",
	"py":   "python",
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
			log.Printf("suffix: %s, name: %s", suffix, name)

			d := map[string]string{
				"chapter": suffixToLang[suffix],
				"page":    "",
				"title":   suffix + ": " + strings.ReplaceAll(name, "-", " "),
				"lang":    suffix,
				"code":    code,
			}

			b, err := yaml.Marshal(d)
			if err != nil {
				log.Fatal(err)
			}

			err = ioutil.WriteFile(fmt.Sprintf(
				"./cmd/play-server/static/examples/gen_%s_%s.yaml",
				suffix, name), b, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
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
