package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
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
	"scala":  "scala",
	"c":      "c_cpp",
	"cc":     "c_cpp",
}

var LangPretty = map[string]string{
	"dotnet": ".NET",
	"java":   "Java",
	"nodejs": "NodeJS",
	"php":    "PHP",
	"py":     "Python",
	"rb":     "Ruby",
	"sh":     "shell",
	"scala":  "Scala",
	"c":      "C",
	"cc":     "C++",
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

	Target target // targeted couchbase

	ExamplesPath string
	Examples     []map[string]interface{} // Sorted.

	Name       string        // Current example name or "".
	Title      template.HTML // Current example title or "".
	Lang       string        // Ex: 'py'.
	LangAce    string        // Ex: 'python'.
	LangPretty string        // Ex: 'Python'.
	Code       string
	Highlight  string
	InfoBefore template.HTML
	InfoAfter  template.HTML

	AnalyticsHTML template.HTML
	OptanonHTML   template.HTML

	PageColor func(string) string

	BodyClass   string
	BaseUrl     string
	FeedbackUrl string
}

func MainTemplateEmit(w http.ResponseWriter,
	staticDir, msg, hostIn string, portApp int,
	version string, versionSDKs []map[string]string,
	session *Session, sessionsMaxAge, sessionsMaxIdle time.Duration,
	Target target, containerPublishPortBase, containerPublishPortSpan int,
	portMapping [][]int,
	examplesPath, name, title, lang, code, highlight, view, bodyClass,
	infoBefore, infoAfter string) error {
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

	// Disable loading the code samples on session to align with new homepage
	// if session != nil && name == "" && lang == "" && code == "" {
	// 	for _, example := range examplesArr {
	// 		if code, exists := example["code"]; exists && code != "" {
	// 			name = example["name"].(string)
	// 			break
	// 		}
	// 	}
	// }

	example, exists := examples[name]
	if exists && example != nil {
		if title == "" {
			title = MapGetString(example, "title")
		}

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

			if session != nil {
				// session couchbase
				if Target.DBurl == "" {
					log.Printf("Session data is getting populated. Target.DBurl is empty")
					code = SessionTemplateExecute(codeHost, portApp, session,
						containerPublishPortBase, containerPublishPortSpan,
						portMapping, code)
				} else {
					log.Println("CBShell only session..no code change should be done.")
					code = TargetTemplateExecute(Target, code, lang)
				}
			} else if Target.DBurl != "" {
				// target couchbase
				code = TargetTemplateExecute(Target, code, lang)
				if strings.Contains(title, "Search Query") {
					CheckAndCreateFtsIndex("travel-fts-index", Target.IPv4, Target.DBuser, Target.DBpwd)
				}
			} else {
				// default non-session couchbase
				code = DefaultTemplateExecute(codeHost, code)
			}

		}

		if infoBefore == "" {
			infoBefore = MapGetString(example, "infoBefore")
		}

		if infoAfter == "" {
			infoAfter = MapGetString(example, "infoAfter")
		}
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

		Target: Target,

		ExamplesPath: examplesPath,
		Examples:     examplesArr,

		Name:       name,
		Title:      template.HTML(title),
		Lang:       lang,
		LangAce:    LangAce[lang],
		LangPretty: LangPretty[lang],
		Code:       code,
		Highlight:  highlight,
		InfoBefore: template.HTML(AddSessionInfo(session, infoBefore)),
		InfoAfter:  template.HTML(infoAfter),

		AnalyticsHTML: template.HTML(AnalyticsHTML(hostIn)),
		OptanonHTML:   template.HTML(OptanonHTML(hostIn)),

		PageColor: func(page string) string { // Ex: {{pageColor "page-02"}}
			n, _ := strconv.Atoi(strings.Split(page, "-")[1])
			n = n - 1
			if n < 0 {
				n = 0
			}

			r := 46 + n*2
			if r > 255 {
				r = 255
			}
			g := 52 + n*2
			if g > 255 {
				g = 255
			}
			b := 64 + n*2
			if b > 255 {
				b = 255
			}

			return fmt.Sprintf("%2x%2x%2x", r, g, b)
		},

		BodyClass:   bodyClass,
		BaseUrl:     *baseUrl,
		FeedbackUrl: *feedbackURL,
	}

	if view != "" {
		view = "-" + view
	}

	// t, err := template.ParseFiles(staticDir + "/main" + view + ".html.tmpl")
	t, err := template.ParseFiles(staticDir + "/home" + view + ".html.tmpl")

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
		"ContainerIP": "",
	}

	if session != nil {
		data["SessionId"] = session.SessionId
		data["CBUser"] = session.CBUser
		data["CBPswd"] = session.CBPswd

		if session.ContainerId >= 0 {
			data["scheme"] = "http"
			if *tlsTerminalProxy {
				containerPublishPortBase = *tlsListenPortBase
				data["scheme"] = "https"
			}
			portBase := containerPublishPortBase +
				(containerPublishPortSpan * session.ContainerId)

			for _, port := range portMapping {
				if port[0] == 8091 {
					if *tlsTerminalProxy {
						data["port_8091"] = fmt.Sprintf("%d", *tlsListenPortBase+port[0])
					} else {
						data["port_8091"] = fmt.Sprintf("%d", port[0])
					}
				} else {
					data[fmt.Sprintf("port_%d", port[0])] = fmt.Sprintf("%d", portBase+port[1])
				}
			}

			data["ContainerId"] = session.ContainerId
		}

		data["ContainerIP"] = session.ContainerIP
	}

	return data
}

func TargetTemplateExecute(Target target, t string, lang string) string {
	data := TargetTemplateData(Target)

	var b bytes.Buffer
	//TBD: better way to get the new code template with couchbases (tls/ssl)
	if &Target != nil && Target.DBurl != "" {
		t = strings.ReplaceAll(t, "couchbase://", "")
		if strings.HasPrefix(Target.DBurl, "couchbases://") {
			if lang == "java" {
				replaceCode := `class Program {`
				secureCode := `
import com.couchbase.client.java.env.ClusterEnvironment;
import com.couchbase.client.core.deps.io.netty.handler.ssl.util.InsecureTrustManagerFactory;
import com.couchbase.client.core.env.IoConfig;
import com.couchbase.client.core.env.SecurityConfig;
import com.couchbase.client.java.ClusterOptions;

class Program {`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				replaceCode = `var cluster = Cluster.connect(`
				secureCode = `ClusterEnvironment env = ClusterEnvironment.builder()
				.securityConfig(SecurityConfig.enableTls(true)
						.trustManagerFactory(InsecureTrustManagerFactory.INSTANCE))
				.ioConfig(IoConfig.enableDnsSrv(true))
				.build();
	Cluster cluster = Cluster.connect("{{.Host}}",`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				replaceCode = `"{{.Host}}", "{{.CBUser}}", "{{.CBPswd}}"`
				secureCode = `ClusterOptions.clusterOptions("{{.CBUser}}", "{{.CBPswd}}").environment(env)`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				data["Host"] = strings.Split(data["Host"].(string), "?")[0] // no ?ssl=no_verify

			} else if lang == "dotnet" {
				replaceCode := `var cluster = await Cluster.ConnectAsync(`
				secureCode := `
	  var opts = new ClusterOptions().WithCredentials("{{.CBUser}}", "{{.CBPswd}}");
	  opts.KvIgnoreRemoteCertificateNameMismatch = true;
	  opts.HttpIgnoreRemoteCertificateMismatch = true;
	  var cluster = await Cluster.ConnectAsync(`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				replaceCode = `"{{.Host}}", "{{.CBUser}}", "{{.CBPswd}}"`
				secureCode = `"{{.Host}}", opts`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				data["Host"] = strings.Split(data["Host"].(string), "?")[0] // no ?ssl=no_verify

			} else if lang == "scala" {
				replaceCode := `object Program extends App {`
				secureCode := `
import com.couchbase.client.scala.env.ClusterEnvironment
import com.couchbase.client.core.deps.io.netty.handler.ssl.util.InsecureTrustManagerFactory
import com.couchbase.client.scala.env.{IoConfig, SecurityConfig, PasswordAuthenticator}
import com.couchbase.client.scala.ClusterOptions

object Program extends App {`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				replaceCode = `val cluster = Cluster.connect("{{.Host}}", "{{.CBUser}}", "{{.CBPswd}}").get`
				secureCode = `
  System.setProperty("com.couchbase.env.security.enableTls", "true")
  val env: ClusterEnvironment = ClusterEnvironment.builder
									.securityConfig(SecurityConfig()
									.trustManagerFactory(InsecureTrustManagerFactory.INSTANCE))
									.build
									.get
  val cluster = Cluster.connect("{{.Host}}",
		ClusterOptions(PasswordAuthenticator("{{.CBUser}}", "{{.CBPswd}}")).environment(env)).get`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				replaceCode = `"{{.Host}}", "{{.CBUser}}", "{{.CBPswd}}"`
				secureCode = `ClusterOptions.clusterOptions("{{.CBUser}}", "{{.CBPswd}}").environment(env)`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				data["Host"] = strings.Split(data["Host"].(string), "?")[0] // no ?ssl=no_verify
				data["Host"] = strings.Split(data["Host"].(string), "//")[1]

			} else if lang == "rb" {
				data["Host"] = strings.Split(data["Host"].(string), "?")[0] // no ?ssl=no_verify
			} else if lang == "sh" {
				replaceCode := `http://{{.CBUser}}:{{.CBPswd}}@{{.Host}}:8093/`
				secureCode := ` -k https://{{.CBUser}}:{{.CBPswd}}@{{.Host}}:18093/`
				t = strings.ReplaceAll(t, replaceCode, secureCode)
				data["Host"] = strings.Split(data["Host"].(string), "?")[0] // no ?ssl=no_verify
				data["Host"] = "cb-0000." + strings.Split(data["Host"].(string), "//")[1]
			}
		}
	}

	err := textTemplate.Must(textTemplate.New("t").Parse(t)).
		Execute(&b, data)
	if err != nil {
		log.Printf("ERROR: TargetTemplateExecute, err: %v", err)

		return t
	}

	return b.String()
}

func TargetTemplateData(Target target) map[string]interface{} {
	data := map[string]interface{}{
		"Host":        host,
		"CBUser":      "username",
		"CBPswd":      "password",
		"NatPublicIP": *natPublicIP,
	}

	if &Target != nil && Target.DBurl != "" && Target.DBuser != "" && Target.DBpwd != "" {
		data["Host"] = Target.DBurl
		data["CBUser"] = Target.DBuser
		data["CBPswd"] = Target.DBpwd
		data["NatPublicIP"] = *natPublicIP
	}

	return data
}

func DefaultTemplateExecute(codeHost string, t string) string {
	data := DefaultTemplateData(codeHost)

	var b bytes.Buffer

	err := textTemplate.Must(textTemplate.New("t").Parse(t)).
		Execute(&b, data)
	if err != nil {
		log.Printf("ERROR: DefaultTemplateExecute, err: %v", err)

		return t
	}

	return b.String()
}

func DefaultTemplateData(codeHost string) map[string]interface{} {
	data := map[string]interface{}{
		"Host":   codeHost,
		"CBUser": "username",
		"CBPswd": "password",
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

		for _, k := range []string{
			"chapter-order", "chapter", "page-order", "page", "title-order", "title",
		} {
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

	var namePrev string

	for _, name := range names {
		examples[name]["name"] = name

		examplesArr = append(examplesArr, examples[name])

		if examples[namePrev] != nil {
			examples[namePrev]["nameNext"] = name
		}

		namePrev = name
	}

	return examples, examplesArr, nil
}
