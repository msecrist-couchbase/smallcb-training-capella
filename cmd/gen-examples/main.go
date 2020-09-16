package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/yaml.v2"
)

func main() {
	m, err := ReadFiles("../sdk-examples/go",
		map[string]bool{
			"go": true,
		}, nil)
	if err != nil {
		log.Fatal(err)
	}

	m, err = ReadFiles("../docs-sdk-java/modules/howtos/examples",
		map[string]bool{
			"java": true,
		}, m)
	if err != nil {
		log.Fatal(err)
	}

	for suffix, m2 := range m {
		for name, code := range m2 {
			log.Printf("suffix: %s, name: %s", suffix, name)

			d := map[string]string{
				"chapter": "go",
				"page":    "",
				"title":   suffix + ": " + strings.ReplaceAll(name, "-", " "),
				"lang":    "go",
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
