package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
)

type NameTitle struct {
	Name, Title string
}

type MainTemplateData struct {
	Examples   string
	NameTitles []NameTitle
	Name       string
	Title      string
	Lang       string // Ex: 'py'.
	Code       string
	InfoBefore string
	InfoAfter  string
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, examplesDir, name, lang, code string) {
	examples, exampleNames, err := ReadExamples(staticDir + "/" + examplesDir)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", ReadExamples, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: ReadExamples, err: %v", err)
		return
	}

	nameTitles := make([]NameTitle, 0, len(exampleNames))
	for _, name := range exampleNames {
		title := MapGetString(examples[name], "title")
		if title == "" {
			title = name
		}

		nameTitles = append(nameTitles, NameTitle{
			Name:  name,
			Title: title,
		})
	}

	var title, infoBefore, infoAfter string

	example, exists := examples[name]
	if exists && example != nil {
		title = MapGetString(example, "title")

		if lang == "" {
			lang = MapGetString(example, "lang")
		}

		if code == "" {
			code = MapGetString(example, "code")
		}

		infoBefore = MapGetString(example, "infoBefore")

		infoAfter = MapGetString(example, "infoAfter")
	}

	data := &MainTemplateData{
		Examples:   examplesDir,
		NameTitles: nameTitles,
		Name:       name,
		Title:      title,
		Lang:       lang,
		Code:       code,
		InfoBefore: infoBefore,
		InfoAfter:  infoAfter,
	}

	t, err := template.ParseFiles(staticDir + "/main.html.template")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", template.ParseFiles, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: template.ParseFiles, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.Execute(w, data)
	if err != nil {
		log.Printf("ERROR: t.Execute, data: %+v, err: %v", data, err)
	}
}

// ------------------------------------------------

// ReadExamples will return...
//   examples:
//     { "basic-py": { "title": "...", "lang": "py", "code": "..." }, ... }.
//   exampleNames (sorted ASC):
//     [ "basic-py", ... ].
func ReadExamples(dir string) (
	examples map[string]map[string]interface{},
	exampleNames []string,
	err error) {
	examples, err = ReadYamls(dir, ".yaml")
	if err != nil {
		return nil, nil, err
	}

	for name, example := range examples {
		// Only yaml's with a title are considered examples.
		if _, hasTitle := example["title"]; hasTitle {
			exampleNames = append(exampleNames, name)
		}
	}

	sort.Strings(exampleNames)

	return examples, exampleNames, nil
}
