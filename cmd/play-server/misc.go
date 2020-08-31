package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

func ReadYamls(dir, suffix string) (
	map[string]map[string]interface{}, error) {
	rv := map[string]map[string]interface{}{}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadDir, dir: %s, err: %v", dir, err)
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), suffix) {
			m, err := ReadYaml(dir + "/" + f.Name())
			if err != nil {
				return nil, fmt.Errorf("ReadYaml, f: %+v, err: %v", f, err)
			}

			rv[f.Name()[:len(f.Name())-len(suffix)]] = m
		}
	}

	return rv, nil
}

func ReadYaml(path string) (map[string]interface{}, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile, path: %s, err: %v", path, err)
	}

	m := make(map[string]interface{})

	err = yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal, path: %s, err: %v", path, err)
	}

	return m, nil
}

// ------------------------------------------------

func MapGetString(m map[string]interface{}, k string) string {
	if v, exists := m[k]; exists {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}
