package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	textTemplate "text/template"
)

type MainTemplateData struct {
	Msg string

	SessionId string // The sessionId can be "".

	ExamplesDir string
	Examples    []ExampleNameTitle

	Name       string // Current example name or "".
	Title      string // Current example title or "".
	Lang       string // Ex: 'py'.
	Code       string
	InfoBefore template.HTML
	InfoAfter  template.HTML
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, msg, sessionId, examplesDir string,
	name, lang, code string, codeData map[string]string) {
	examples, exampleNameTitles, err :=
		ReadExamples(staticDir + "/" + examplesDir)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", ReadExamples, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: ReadExamples, err: %v", err)
		return
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

			code = CodeTemplateExecute(code, codeData)
		}

		infoBefore = MapGetString(example, "infoBefore")

		infoAfter = MapGetString(example, "infoAfter")
	}

	data := &MainTemplateData{
		Msg: msg,

		SessionId: sessionId,

		ExamplesDir: examplesDir,
		Examples:    exampleNameTitles,

		Name:       name,
		Title:      title,
		Lang:       lang,
		Code:       code,
		InfoBefore: template.HTML(infoBefore),
		InfoAfter:  template.HTML(infoAfter),
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
		log.Printf("ERROR: t.Execute, err: %v", err)
	}
}

// ------------------------------------------------

func CodeTemplateExecute(code string, codeData map[string]string) string {
	if codeData == nil {
		codeData = map[string]string{}
	}

	if codeData["cbUser"] == "" {
		codeData["cbUser"] = "Administrator"
	}

	if codeData["cbPswd"] == "" {
		codeData["cbPswd"] = "password"
	}

	var b bytes.Buffer

	err := textTemplate.Must(textTemplate.New("code").Parse(code)).
		Execute(&b, codeData)
	if err != nil {
		log.Printf("ERROR: CodeTemplateExecute, err: %v", err)

		return code
	}

	return b.String()
}

// ------------------------------------------------

type ExampleNameTitle struct {
	Name, Title string
}

// ReadExamples will return...
//   examples:
//     { "basic-py": { "title": "...", "lang": "py", "code": "..." }, ... }.
//   exampleNames (sorted ASC):
//     [ "basic-py", ... ].
func ReadExamples(dir string) (
	examples map[string]map[string]interface{},
	exampleNameTitles []ExampleNameTitle, err error) {
	examples, err = ReadYamls(dir, ".yaml")
	if err != nil {
		return nil, nil, err
	}

	names := make([]string, 0, len(examples))
	for name, example := range examples {
		// Only yaml's with a title are considered examples.
		if _, hasTitle := example["title"]; hasTitle {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	for _, name := range names {
		exampleNameTitles = append(exampleNameTitles, ExampleNameTitle{
			Name:  name,
			Title: MapGetString(examples[name], "title"),
		})
	}

	return examples, exampleNameTitles, nil
}
