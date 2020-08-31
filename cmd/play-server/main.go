package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

var (
	h = flag.Bool("h", false, "print help/usage and exit")

	help = flag.Bool("help", false, "print help/usage and exit")

	codeMaxLen = flag.Int("codeMaxLen", 16000,
		"max length of a client's request code in bytes")

	codeDuration = flag.Duration("codeDuration", 10*time.Second,
		"duration that a client's request code may run on an assigned container instance")

	containerNamePrefix = flag.String("containerNamePrefix", "smallcb-",
		"prefix of the names of container instances")

	containerVolPrefix = flag.String("containerVolPrefix", "vol-",
		"prefix of the volume directories of container instances")

	containerPublishAddr = flag.String("containerPublishAddr", "127.0.0.1",
		"addr for publishing container instance ports")

	containerPublishPortBase = flag.Int("containerPublishPortBase", 10000,
		"base or starting port # for container instances")

	containerPublishPortSpan = flag.Int("containerPublishPortSpan", 100,
		"number of port #'s allocated for each container instance")

	containerWaitDuration = flag.Duration("containerWaitDuration", 20*time.Second,
		"duration that a client's request will wait for a ready container instance")

	containers = flag.Int("containers", 1,
		"# of container instances")

	restarters = flag.Int("restarters", 1,
		"# of restarters of the container instances")

	static = flag.String("static", "cmd/play-server/static",
		"path to the 'static' resources directory")

	listen = flag.String("listen", ":8080",
		"HTTP listen [addr]:port")

	// -----------------------------------

	TitleDefault = "API / SDK Playground"

	// -----------------------------------

	containersCh chan int // Channel of container instance #'s that are ready.

	restarterCh chan int // Channel of container instance #'s that need restart.

	// -----------------------------------

	// The langs is a config table with entries of...
	//   [ lang (code file suffix),
	//     langName (for UI display),
	//     execPrefix (exec command prefix, if any, for executing code) ].
	langs = [][]string{
		[]string{"java", "java", "/run-java.sh"},
		[]string{"py", "python3", ""},
	}

	langNames = map[string]string{} // Map from 'py' to 'python3'.
	langExecs = map[string]string{} // Map from 'py' to execPrefix.

	// -----------------------------------

	// Port mapping of container port # to containerPublishPortBase + delta.
	portMapping = [][]int{
		[]int{8091, 1}, // 8091 is exposed on port 10000 + 1.
		[]int{8092, 2}, // 8092 is exposed on port 10000 + 2.
		[]int{8093, 3},
		[]int{8094, 4},
		[]int{8095, 5},
		[]int{8096, 6},

		[]int{18091, 11}, // 18091 is exposed on port 10000 + 11.
		[]int{18092, 12}, // 18092 is exposed on port 10000 + 12.
		[]int{18093, 13},
		[]int{18094, 14},
		[]int{18095, 15},
		[]int{18096, 16},

		[]int{11207, 27}, // 11207 is exposed on port 10000 + 27.
		[]int{11210, 30}, // 11210 is exposed on port 10000 + 30.
		[]int{11211, 31}, // 11211 is exposed on port 10000 + 31.
	}
)

// ------------------------------------------------

func init() {
	for _, item := range langs {
		lang, langName, langExec := item[0], item[1], item[2]

		langNames[lang] = langName
		langExecs[lang] = langExec
	}
}

// ------------------------------------------------

func main() {
	flag.Parse()

	if *h || *help {
		flag.Usage()
		os.Exit(2)
	}

	// The containersCh and restarterCh are created with capacity
	// equal to the # of containers to lower the chance of
	// client requests and restarters from having to wait.

	containersCh = make(chan int, *containers)

	restarterCh = make(chan int, *containers)

	// Spawn the restarter goroutines.
	for i := 0; i < *restarters; i++ {
		go Restarter(i, restarterCh, containersCh,
			*containerPublishAddr,
			*containerPublishPortBase,
			*containerPublishPortSpan,
			portMapping)
	}

	// Have the restarters restart the required # of containers.
	for containerId := 0; containerId < *containers; containerId++ {
		restarterCh <- containerId
	}

	mux := http.NewServeMux()

	HttpMuxInit(mux)

	log.Printf("INFO: listening on... %v", *listen)

	log.Fatal(http.ListenAndServe(*listen, mux))
}

// ------------------------------------------------

func HttpMuxInit(mux *http.ServeMux) {
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*static))))

	mux.HandleFunc("/run", HttpHandleRun)

	mux.HandleFunc("/", HttpHandleMain)
}

// ------------------------------------------------

func HttpHandleMain(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	if strings.HasPrefix(r.URL.Path, "/example/") &&
		len(r.URL.Path) > len("/example/") {
		name = r.URL.Path[len("/example/"):]
	}

	lang := r.FormValue("lang")
	code := r.FormValue("code")

	MainTemplateEmit(w, *static, *static+"/examples", name, lang, code)
}

// ------------------------------------------------

func HttpHandleRun(w http.ResponseWriter, r *http.Request) {
	lang := r.FormValue("lang")
	code := r.FormValue("code")

	output, err := RunLangCode(r.Context(), lang, code,
		*codeMaxLen, *codeDuration,
		containersCh,
		*containerWaitDuration,
		*containerNamePrefix,
		*containerVolPrefix,
		restarterCh)
	if err != nil {
		http.Error(w,
			http.StatusText(http.StatusInternalServerError)+
				fmt.Sprintf(", RunLangCode, err: %v", err),
			http.StatusInternalServerError)
		log.Printf("ERROR: RunLangCode, err: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.Write([]byte(output))
}

// ------------------------------------------------

type NameTitle struct {
	Name, Title string
}

type MainTemplateData struct {
	NameTitles []NameTitle
	Name       string
	Title      string
	Lang       string // Ex: 'py'.
	Code       string
	Output     string
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, examplesDir, name, lang, code string) {
	examples, exampleNames, err := ReadExamples(examplesDir)
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
		title, _ := examples[name]["title"].(string)
		if title == "" {
			title = name
		}

		nameTitles = append(nameTitles, NameTitle{
			Name:  name,
			Title: title,
		})
	}

	if name == "" && lang == "" && code == "" {
		name = "basic-py"
	}

	var title string

	example := examples[name]
	if example != nil {
		if title == "" {
			title = example["title"].(string)
		}

		if lang == "" {
			lang = example["lang"].(string)
		}

		if code == "" {
			code = example["code"].(string)
		}
	}

	if title == "" {
		title = TitleDefault
	}

	data := &MainTemplateData{
		NameTitles: nameTitles,
		Name:       name,
		Title:      title,
		Lang:       lang,
		Code:       code,
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

// ------------------------------------------------
