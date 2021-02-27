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

// Map from lang suffix to name that ACE editor recognizes.
var LangAce = map[string]string{
	"go":     "golang",
	"java":   "java",
	"nodejs": "javascript",
	"php":    "php",
	"py":     "python",
	"rb":     "ruby",
	"sh":     "sh",
	"dotnet": "csharp",
}

var LangPretty = map[string]string{
	"dotnet": ".NET",
	"java":   "Java",
	"nodejs": "NodeJS",
	"php":    "PHP",
	"py":     "Python",
	"rb":     "Ruby",
	"sh":     "shell",
}

type MainTemplateData struct {
	Msg string

	Host string

	Version     string
	VersionSDKs []map[string]string

	Session         *Session // May be nil.
	SessionData     map[string]interface{}
	SessionsMaxAge  string
	SessionsMaxIdle string

	ExamplesPath string
	Examples     []map[string]interface{} // Sorted.

	Name       string // Current example name or "".
	Title      string // Current example title or "".
	Lang       string // Ex: 'py'.
	LangAce    string // Ex: 'python'.
	LangPretty string // Ex: 'Python'.
	Code       string
	InfoBefore template.HTML
	InfoAfter  template.HTML

	AnalyticsHTML template.HTML
	OptanonHTML   template.HTML
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, msg, hostIn string, portApp int,
	version string, versionSDKs []map[string]string,
	session *Session, sessionsMaxAge, sessionsMaxIdle time.Duration,
	containerPublishPortBase, containerPublishPortSpan int,
	portMapping [][]int,
	examplesPath string, name, lang, code, view string) error {
	host := hostIn
	if session == nil {
		host = "127.0.0.1"
	}

	examples, examplesArr, err :=
		ReadExamples(staticDir + "/" + examplesPath)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", ReadExamples, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: ReadExamples, err: %v", err)
		return err
	}

	var title, infoBefore, infoAfter string

	if session != nil && name == "" && lang == "" && code == "" {
		for _, example := range examplesArr {
			if code, exists := example["code"]; exists && code != "" {
				name = example["name"].(string)
				break
			}
		}
	}

	example, exists := examples[name]
	if exists && example != nil {
		title = MapGetString(example, "title")

		if lang == "" {
			lang = MapGetString(example, "lang")
		}

		if code == "" {
			code = MapGetString(example, "code")

			// TODO: NOTE: Java & .NET SDK's can't seem to
			// use a proper host, so for now all code
			// samples will use 127.0.0.1 no matter if
			// there's a session or not.
			codeHost := "127.0.0.1" // host.

			code = SessionTemplateExecute(codeHost, portApp, session,
				containerPublishPortBase, containerPublishPortSpan,
				portMapping, code)
		}

		infoBefore = MapGetString(example, "infoBefore")

		infoAfter = MapGetString(example, "infoAfter")
	}

	data := &MainTemplateData{
		Msg: msg,

		Host: host,

		Version:     version,
		VersionSDKs: versionSDKs,

		Session: session,

		SessionData: SessionTemplateData(host, portApp, session,
			containerPublishPortBase, containerPublishPortSpan, portMapping),

		SessionsMaxAge: strings.Replace(
			sessionsMaxAge.String(), "m0s", " min", 1),

		SessionsMaxIdle: strings.Replace(
			sessionsMaxIdle.String(), "m0s", " min", 1),

		ExamplesPath: examplesPath,
		Examples:     examplesArr,

		Name:       name,
		Title:      title,
		Lang:       lang,
		LangAce:    LangAce[lang],
		LangPretty: LangPretty[lang],
		Code:       code,
		InfoBefore: template.HTML(infoBefore),
		InfoAfter:  template.HTML(infoAfter),

		AnalyticsHTML: template.HTML(AnalyticsHTML(hostIn)),
		OptanonHTML:   template.HTML(OptanonHTML(hostIn)),
	}

	if view != "" {
		view = "-" + view
	}

	t, err := template.ParseFiles(staticDir + "/main" + view + ".html.tmpl")
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", template.ParseFiles, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: template.ParseFiles, err: %v", err)
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = t.Execute(w, data)
	if err != nil {
		log.Printf("ERROR: t.Execute, err: %v", err)
	}

	return err
}

// ------------------------------------------------

func SessionTemplateExecute(host string, portApp int,
	session *Session,
	containerPublishPortBase, containerPublishPortSpan int,
	portMapping [][]int,
	t string) string {
	data := SessionTemplateData(host, portApp, session,
		containerPublishPortBase, containerPublishPortSpan, portMapping)

	var b bytes.Buffer

	err := textTemplate.Must(textTemplate.New("t").Parse(t)).
		Execute(&b, data)
	if err != nil {
		log.Printf("ERROR: SessionTemplateExecute, err: %v", err)

		return t
	}

	return b.String()
}

func SessionTemplateData(host string, portApp int,
	session *Session,
	containerPublishPortBase, containerPublishPortSpan int,
	portMapping [][]int) map[string]interface{} {
	data := map[string]interface{}{
		"Host":        host,
		"PortApp":     fmt.Sprintf("%d", portApp),
		"CBUser":      "username",
		"CBPswd":      "password",
		"ContainerId": -1,
	}

	if session != nil {
		data["SessionId"] = session.SessionId
		data["CBUser"] = session.CBUser
		data["CBPswd"] = session.CBPswd

		if session.ContainerId >= 0 {
			portBase := containerPublishPortBase +
				(containerPublishPortSpan * session.ContainerId)

			for _, port := range portMapping {
				data[fmt.Sprintf("port_%d", port[0])] = fmt.Sprintf("%d", portBase+port[1])
			}

			data["ContainerId"] = session.ContainerId
		}
	}

	return data
}

// ------------------------------------------------

// ReadExamples will return...
//   examples:
//     { "basic-py": { "title": "...", "lang": "py", "code": "..." }, ... }.
//   exampleNames (sorted ASC):
//     [ "basic-py", ... ].
func ReadExamples(dir string) (
	examples map[string]map[string]interface{},
	examplesArr []map[string]interface{}, err error) {
	examples, err = ReadYamls(dir, ".yaml", nil)
	if err != nil {
		return nil, nil, err
	}

	names := make([]string, 0, len(examples))
	for name, example := range examples {
		// Only yaml's with a title are considered examples.
		if MapGetString(example, "title") != "" {
			names = append(names, name)
		}
	}

	sort.Slice(names, func(i, j int) bool {
		iname, jname := names[i], names[j]

		iex, jex := examples[iname], examples[jname]

		for _, k := range []string{"chapter", "page", "title"} {
			iv := MapGetString(iex, k)
			jv := MapGetString(jex, k)

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
		examples[name]["name"] = name
		examplesArr = append(examplesArr, examples[name])
	}

	return examples, examplesArr, nil
}
