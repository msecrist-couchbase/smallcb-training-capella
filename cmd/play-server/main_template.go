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

type NameTitle struct {
	Name, Title string
}

type MainTemplateData struct {
	SessionId  string      // The sessionId can be "".
	Examples   string      // The examplesDir, like "examples".
	NameTitles []NameTitle // The example names & titles from the examplesDir.
	Name       string      // Current example name or "".
	Title      string      // Current example title or "".
	Lang       string      // Ex: 'py'.
	Code       string
	InfoBefore template.HTML
	InfoAfter  template.HTML
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, sessionId, examplesDir string,
	name, lang, code string, codeData map[string]string) {
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

			code = CodeTemplateExecute(code, codeData)
		}

		infoBefore = MapGetString(example, "infoBefore")

		infoAfter = MapGetString(example, "infoAfter")
	}

	data := &MainTemplateData{
		SessionId:  sessionId,
		Examples:   examplesDir,
		NameTitles: nameTitles,
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
		log.Printf("ERROR: t.Execute, data: %+v, err: %v", data, err)
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
		log.Printf("ERROR: textTemplate.Execute, codeData: %+v, err: %v", codeData, err)

		return code
	}

	return b.String()
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
