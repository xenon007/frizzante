package frizzante

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"os"
	"path/filepath"
	"rogchap.com/v8go"
	"strings"
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

	renderClientSvelte, renderClientSpaSvelteError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.client.svelte")
	if renderClientSpaSvelteError != nil {
		panic(renderClientSpaSvelteError)
	}

	renderServerJs, renderServerJsError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.server.js")
	if renderServerJsError != nil {
		panic(renderServerJsError)
	}

	renderServerSvelte, renderServerSvelteError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.server.svelte")
	if renderServerSvelteError != nil {
		panic(renderServerSvelteError)
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

	err = os.WriteFile("www/.frizzante/vite-project/render.client.svelte", renderClientSvelte, os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("www/.frizzante/vite-project/render.server.js", renderServerJs, os.ModePerm)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("www/.frizzante/vite-project/render.server.svelte", renderServerSvelte, os.ModePerm)
	if err != nil {
		panic(err)
	}

}

var sveltePages = map[string]string{}

func SveltePreparePage(id string, fileName string) {
	relativeFileName, err := filepath.Rel("www/.frizzante/vite-project", fileName)
	if err != nil {
		panic(err)
	}
	sveltePages[id] = fmt.Sprintf("./%s", relativeFileName)
}

func SveltePrepareEnd() {
	if !Exists("www/.frizzante/vite-project") {
		err := os.MkdirAll("www/.frizzante/vite-project", os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	// Build client loader.
	var builder strings.Builder
	builder.WriteString("<script>\n")
	builder.WriteString("    import Async from './async.svelte'\n")
	builder.WriteString("    let {page, ...data} = $props()\n")
	builder.WriteString("</script>\n")
	counter := 0
	for id, fileName := range sveltePages {
		if 0 == counter {
			builder.WriteString(fmt.Sprintf("{#if '%s' === page}\n", id))
		} else {
			builder.WriteString(fmt.Sprintf("{:else if '%s' === page}\n", id))
		}
		builder.WriteString(fmt.Sprintf("    <Async from={import('%s')} {...data} />\n", fileName))
		counter++
	}
	if counter > 0 {
		builder.WriteString("{/if}")
	}
	renderClientSvelte := builder.String()

	// Dump client loader.
	err := os.WriteFile("www/.frizzante/vite-project/render.client.svelte", []byte(renderClientSvelte), os.ModePerm)
	if err != nil {
		panic(err)
	}

	// Build server loader.
	builder.Reset()
	builder.WriteString("<script>\n")
	for id, fileName := range sveltePages {
		builder.WriteString(fmt.Sprintf("    import %s from '%s'\n", strings.ToUpper(id), fileName))
	}
	builder.WriteString("    let {page, ...data} = $props()\n")
	builder.WriteString("</script>\n")
	counter = 0
	for id, _ := range sveltePages {
		if 0 == counter {
			builder.WriteString(fmt.Sprintf("{#if '%s' === page}\n", id))
		} else {
			builder.WriteString(fmt.Sprintf("{:else if '%s' === page}\n", id))
		}
		builder.WriteString(fmt.Sprintf("    <%s {...data} />\n", strings.ToUpper(id)))
		counter++
	}
	if counter > 0 {
		builder.WriteString("{/if}")
	}
	renderServerSvelte := builder.String()

	// Dump server loader.
	err = os.WriteFile("www/.frizzante/vite-project/render.server.svelte", []byte(renderServerSvelte), os.ModePerm)
	if err != nil {
		panic(err)
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

		globals[key] = value
	}

	_, destroy, javaScriptError := JavaScriptRun(doneCjs, allGlobals)
	if javaScriptError != nil {
		return head, body, javaScriptError
	}
	defer destroy()

	return head, body, nil
}

type SveltePageOptions struct {
	Ssr     bool
	Props   map[string]interface{}
	Globals map[string]v8go.FunctionCallback
}

// SveltePage renders and echos a svelte page.
func SveltePage(response *Response, options *SveltePageOptions) {
	var optionsLocal *SveltePageOptions

	if nil == options {
		optionsLocal = &SveltePageOptions{
			Ssr:     true,
			Props:   map[string]interface{}{},
			Globals: map[string]v8go.FunctionCallback{},
		}
	} else {
		optionsLocal = options
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

	bytesProps, jsonError := json.Marshal(optionsLocal.Props)
	if jsonError != nil {
		ServerNotifyError(response.server, jsonError)
		return
	}
	stringProps := string(bytesProps)

	head := ""
	body := ""
	if optionsLocal.Ssr {
		headLocal, bodyLocal, renderError := render(response, stringProps, optionsLocal.Globals)
		if renderError != nil {
			ServerNotifyError(response.server, renderError)
			return
		}
		head = headLocal
		body = bodyLocal
	}

	stringIndex := strings.Replace(
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
		"<!--app-props-->",
		fmt.Sprintf(
			`<script type="application/javascript">function props(){return %s}</script>`,
			stringProps,
		),
		1,
	)

	Header(response, "Content-Type", "text/html")
	Echo(response, stringIndex)
}
