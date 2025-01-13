package frizzante

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"os"
	"path/filepath"
	"regexp"
	"rogchap.com/v8go"
	"strings"
)

type RenderMode int64

const (
	ModeServer RenderMode = 0 // Render only on the server.
	ModeClient RenderMode = 1 // Render only on the client.
	ModeFull   RenderMode = 2 // Render on both the server and the client.
)

//go:embed vite-project/*
var svelteRenderToolsFileSystem embed.FS

func SveltePrepareStart() {
	asyncSvelte, asyncSvelteError := svelteRenderToolsFileSystem.ReadFile("vite-project/async.svelte")
	if asyncSvelteError != nil {
		panic(asyncSvelteError)
	}

	indexHtml, indexHtmlError := svelteRenderToolsFileSystem.ReadFile("vite-project/index.html")
	if indexHtmlError != nil {
		panic(indexHtmlError)
	}

	renderClientJs, renderClientJsError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.client.js")
	if renderClientJsError != nil {
		panic(renderClientJsError)
	}

	renderServerJs, renderServerJsError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.server.js")
	if renderServerJsError != nil {
		panic(renderServerJsError)
	}

	if !Exists("www/.frizzante/vite-project") {
		err := os.MkdirAll("www/.frizzante/vite-project", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	err := os.WriteFile("www/.frizzante/vite-project/async.svelte", asyncSvelte, os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("www/.frizzante/vite-project/index.html", indexHtml, os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("www/.frizzante/vite-project/render.client.js", renderClientJs, os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("www/.frizzante/vite-project/render.server.js", renderServerJs, os.ModePerm)
	if err != nil {
		panic(err)
	}
}

var sveltePagesToFileNames = map[string]string{}

func SveltePreparePage(id string, fileName string) {
	relativeFileName, err := filepath.Rel("www/.frizzante/vite-project", fileName)
	if err != nil {
		panic(err)
	}
	sveltePagesToFileNames[id] = fmt.Sprintf("./%s", relativeFileName)
}

func SveltePrepareEnd() {
	if !Exists("www/.frizzante/vite-project") {
		err := os.MkdirAll("www/.frizzante/vite-project", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	// Build client loader.
	renderClientSvelte, readError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.client.svelte")
	if readError != nil {
		panic(readError)
	}

	var builder strings.Builder
	counter := 0
	for id, fileName := range sveltePagesToFileNames {
		if 0 == counter {
			builder.WriteString(fmt.Sprintf("{#if '%s' === reactivePageId}\n", id))
		} else {
			builder.WriteString(fmt.Sprintf("{:else if '%s' === reactivePageId}\n", id))
		}
		builder.WriteString(fmt.Sprintf("    <Async from={import('%s')} />\n", fileName))
		counter++
	}
	if counter > 0 {
		builder.WriteString("{/if}")
	}
	renderClientSvelteString := strings.Replace(string(renderClientSvelte), "<!--app-router-->", builder.String(), 1)

	// Dump client loader.
	marshalError := os.WriteFile("www/.frizzante/vite-project/render.client.svelte", []byte(renderClientSvelteString), os.ModePerm)
	if marshalError != nil {
		panic(marshalError)
	}

	// Build server loader.
	builder.Reset()
	builder.WriteString("<script>\n")
	for id, fileName := range sveltePagesToFileNames {
		builder.WriteString(fmt.Sprintf("    import %s from '%s'\n", strings.ToUpper(id), fileName))
	}
	builder.WriteString("    import {setContext} from 'svelte'\n")
	builder.WriteString("    let {pageId, pagesToPaths, ...data} = $props()\n")
	builder.WriteString("    let reactiveData = $state({...data})\n")
	builder.WriteString("    setContext(\"data\", reactiveData)\n")
	builder.WriteString("    setContext(\"page\", page)\n")
	builder.WriteString("    function page(){}\n")
	builder.WriteString("</script>\n")
	counter = 0
	for id, _ := range sveltePagesToFileNames {
		if 0 == counter {
			builder.WriteString(fmt.Sprintf("{#if '%s' === pageId}\n", id))
		} else {
			builder.WriteString(fmt.Sprintf("{:else if '%s' === pageId}\n", id))
		}
		builder.WriteString(fmt.Sprintf("    <%s />\n", strings.ToUpper(id)))
		counter++
	}
	if counter > 0 {
		builder.WriteString("{/if}")
	}
	renderServerSvelte := builder.String()

	// Dump server loader.
	marshalError = os.WriteFile("www/.frizzante/vite-project/render.server.svelte", []byte(renderServerSvelte), os.ModePerm)
	if marshalError != nil {
		panic(marshalError)
	}
}

func render(response *Response, stringProps string, globals map[string]v8go.FunctionCallback) (string, string, error) {
	renderFileName := filepath.Join("www", "dist", "server", "render.server.js")

	var renderEsmBytes []byte
	if "1" == os.Getenv("DEV") {
		renderEsmBytesLocal, readError := os.ReadFile(renderFileName)
		if readError != nil {
			return "", "", readError
		}
		renderEsmBytes = renderEsmBytesLocal
	} else {
		renderEsmBytesLocal, readError := response.server.embeddedFileSystem.ReadFile(renderFileName)
		if readError != nil {
			return "", "", readError
		}
		renderEsmBytes = renderEsmBytesLocal
	}

	renderEsm := string(renderEsmBytes)

	renderCjs, javaScriptBundleError := JavaScriptBundle("www", api.FormatCommonJS, renderEsm)
	if javaScriptBundleError != nil {
		return "", "", javaScriptBundleError
	}

	renderIif := fmt.Sprintf("const module={exports:{}}; const render = \n(function(){\n%s\nreturn render;\n})()", renderCjs)

	doneEsm := fmt.Sprintf(
		`
		%s
		render(%s).then(function done(rendered){
			head(rendered.head??'');
			body(rendered.body??'');
		});
		`,
		renderIif,
		stringProps,
	)

	doneCjs, bundleError := JavaScriptBundle("www", api.FormatCommonJS, doneEsm)
	if bundleError != nil {
		return "", "", bundleError
	}

	head := ""
	body := ""
	allGlobals := map[string]v8go.FunctionCallback{
		"inspect": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				message := args[0].String()
				println(message)
			}
			return nil
		},
		"head": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				head = args[0].String()
			}
			return nil
		},
		"body": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				body = args[0].String()
			}
			return nil
		},
	}

	for key, value := range globals {
		_, exists := globals[key]
		if exists {
			continue
		}

		allGlobals[key] = value
	}

	_, destroy, javaScriptError := JavaScriptRun(doneCjs, allGlobals)
	if javaScriptError != nil {
		return head, body, javaScriptError
	}
	defer destroy()

	return head, body, nil
}

type SveltePageConfiguration struct {
	Render  RenderMode
	Props   map[string]interface{}
	Globals map[string]v8go.FunctionCallback
}

var noScriptPattern = regexp.MustCompile(`<script.*>.*</script>`)

// EchoSveltePage renders and echos a svelte page.
func EchoSveltePage(response *Response, configuration *SveltePageConfiguration) {
	if nil == configuration {
		configuration = &SveltePageConfiguration{
			Render:  ModeFull,
			Props:   map[string]interface{}{},
			Globals: map[string]v8go.FunctionCallback{},
		}
	}

	fileNameIndex := filepath.Join("www", "dist", "client", ".frizzante", "vite-project", "index.html")

	var indexBytes []byte

	if "1" == os.Getenv("DEV") {
		indexBytesLocal, readError := os.ReadFile(fileNameIndex)
		if readError != nil {
			ServerNotifyError(response.server, readError)
			return
		}
		indexBytes = indexBytesLocal
	} else {
		indexBytesLocal, readError := response.server.embeddedFileSystem.ReadFile(fileNameIndex)
		if readError != nil {
			ServerNotifyError(response.server, readError)
			return
		}
		indexBytes = indexBytesLocal
	}

	bytesProps, jsonError := json.Marshal(configuration.Props)
	if jsonError != nil {
		ServerNotifyError(response.server, jsonError)
		return
	}
	stringProps := string(bytesProps)

	var index string
	if ModeFull == configuration.Render {
		head, body, renderError := render(response, stringProps, configuration.Globals)
		if renderError != nil {
			ServerNotifyError(response.server, renderError)
			return
		}
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						`<script type="application/javascript">function target(){return document.getElementById("app")}</script>`,
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="app">%s</div>`, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function data(){return %s}</script>`,
				stringProps,
			),
			1,
		)
	} else if ModeClient == configuration.Render {
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						`<script type="application/javascript">function target(){return document.getElementById("app")}</script>`,
						1,
					),
					"<!--app-body-->",
					`<div id="app"></div>`,
					1,
				),
				"<!--app-head-->",
				"",
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function data(){return %s}</script>`,
				stringProps,
			),
			1,
		)
	} else if ModeServer == configuration.Render {
		head, body, renderError := render(response, stringProps, configuration.Globals)
		if renderError != nil {
			ServerNotifyError(response.server, renderError)
			return
		}
		index = strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						noScriptPattern.ReplaceAllString(string(indexBytes), ""),
						"<!--app-target-->",
						``,
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="app">%s</div>`, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			"",
			1,
		)
	}

	SendHeader(response, "Content-Type", "text/html")
	SendEcho(response, index)
}
