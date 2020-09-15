package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	textTemplate "text/template"
	"time"
)

type MainTemplateData struct {
	Msg string

	Host string

	Version string

	Session        *Session // May be nil.
	SessionsMaxAge string

	ExamplesPath string
	Examples     []ExampleNameTitle

	Name       string // Current example name or "".
	Title      string // Current example title or "".
	Lang       string // Ex: 'py'.
	Code       string
	InfoBefore template.HTML
	InfoAfter  template.HTML
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, msg, host string, portApp int, version string,
	session *Session, sessionsMaxAge time.Duration,
	examplesPath string, name, lang, code string) {
	examples, exampleNameTitles, err :=
		ReadExamples(staticDir + "/" + examplesPath)
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

			code = SessionTemplateExecute(host, portApp, session, code)
		}

		infoBefore = MapGetString(example, "infoBefore")

		infoAfter = MapGetString(example, "infoAfter")
	}

	data := &MainTemplateData{
		Msg: msg,

		Host: host,

		Version: version,

		Session: session,

		SessionsMaxAge: strings.Replace(
			sessionsMaxAge.String(), "m0s", " min", 1),

		ExamplesPath: examplesPath,
		Examples:     exampleNameTitles,

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

func SessionTemplateExecute(host string, portApp int, session *Session, t string) string {
	data := SessionTemplateData(host, portApp, session)

	var b bytes.Buffer

	err := textTemplate.Must(textTemplate.New("t").Parse(t)).
		Execute(&b, data)
	if err != nil {
		log.Printf("ERROR: SessionTemplateExecute, err: %v", err)

		return t
	}

	return b.String()
}

func SessionTemplateData(host string, portApp int, session *Session) map[string]interface{} {
	data := map[string]interface{}{
		"Host":    host,
		"PortApp": fmt.Sprintf("%d", portApp),
		"CBUser":  "username",
		"CBPswd":  "password",
	}

	if session != nil {
		data["SessionId"] = session.SessionId
		data["CBUser"] = session.CBUser
		data["CBPswd"] = session.CBPswd
	}

	return data
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
		if MapGetString(example, "title") != "" &&
			MapGetString(example, "display") != "none" {
			names = append(names, name)
		}
	}

	sort.Slice(names, func(i, j int) bool {
		iname, jname := names[i], names[j]

		iex, jex := examples[iname], examples[jname]

		for _, k := range []string{"chapter", "page", "title"} {
			iv, jv := MapGetString(iex, k), MapGetString(jex, k)
			if iv < jv {
				return true
			}
			if iv > jv {
				return false
			}
		}

		if iname < jname {
			return true
		}

		return false
	})

	for _, name := range names {
		exampleNameTitles = append(exampleNameTitles, ExampleNameTitle{
			Name:  name,
			Title: MapGetString(examples[name], "title"),
		})
	}

	return examples, exampleNameTitles, nil
}
