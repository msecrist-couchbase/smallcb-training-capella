package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

func ReadYamls(dir, suffix string, rv map[string]map[string]interface{}) (
	map[string]map[string]interface{}, error) {
	if rv == nil {
		rv = map[string]map[string]interface{}{}
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadDir, dir: %s, err: %v", dir, err)
	}

	for _, f := range files {
		if f.IsDir() {
			rv, err = ReadYamls(dir+"/"+f.Name(), suffix, rv)
			if err != nil {
				return nil, err
			}

			continue
		}

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

// Converts map[interface{}]interface{} to
// map[string]interface{}, which is friendlier for JSON
// marshaling -- see: https://github.com/go-yaml/yaml/issues/139

func CleanupInterfaceValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, vv := range v {
			res[i] = CleanupInterfaceValue(vv)
		}
		return res
	case map[interface{}]interface{}:
		res := make(map[string]interface{})
		for k, vv := range v {
			res[fmt.Sprintf("%v", k)] = CleanupInterfaceValue(vv)
		}
		return res
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, vv := range v {
			res[fmt.Sprintf("%v", k)] = CleanupInterfaceValue(vv)
		}
		return res
	case map[string]map[string]interface{}:
		res := make(map[string]interface{})
		for k, vv := range v {
			res[fmt.Sprintf("%v", k)] = CleanupInterfaceValue(vv)
		}
		return res
	case string:
		return v
	default:
		fmt.Printf("unknown type: %T %#v\n", v, v)

		return fmt.Sprintf("%v", v)
	}
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
