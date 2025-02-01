package frizzante

import (
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	uuid "github.com/nu7hatch/gouuid"
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

func render(response *Response, stringProps string, globals map[string]v8go.FunctionCallback) (string, string, error) {
	renderFileName := filepath.Join(".dist", "server", "render.server.js")

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

	renderCjs, javaScriptBundleError := JavaScriptBundle(".", api.FormatCommonJS, renderEsm)
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

	doneCjs, bundleError := JavaScriptBundle(".", api.FormatCommonJS, doneEsm)
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
	Data    map[string]interface{}
	Globals map[string]v8go.FunctionCallback
}

var sveltePagesToPaths = map[string]string{}

type svelteRouterProps struct {
	PageId string                 `json:"pageId"`
	Data   map[string]interface{} `json:"data"`
	Paths  map[string]string      `json:"paths"`
}

var noScriptPattern = regexp.MustCompile(`<script.*>.*</script>`)

// SendSveltePage renders and echos a svelte page.
func SendSveltePage(
	response *Response,
	pageId string,
	configuration *SveltePageConfiguration,
) {
	if nil == configuration {
		configuration = &SveltePageConfiguration{
			Render:  ModeFull,
			Data:    map[string]interface{}{},
			Globals: map[string]v8go.FunctionCallback{},
		}
	}

	fileNameIndex := filepath.Join(".dist", "client", ".frizzante", "vite-project", "index.html")

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

	routerPropsBytes, jsonError := json.Marshal(svelteRouterProps{
		PageId: pageId,
		Data:   configuration.Data,
		Paths:  sveltePagesToPaths,
	})
	if jsonError != nil {
		ServerNotifyError(response.server, jsonError)
		return
	}
	routerPropsString := string(routerPropsBytes)

	targetId, targetIdError := uuid.NewV4()
	if targetIdError != nil {
		panic(targetIdError)
	}

	var index string
	if ModeFull == configuration.Render {
		head, body, renderError := render(response, routerPropsString, configuration.Globals)
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
						fmt.Sprintf(`<script type="application/javascript">function target(){return document.getElementById("%s")}</script>`, targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="%s">%s</div>`, targetId, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function props(){return %s}</script>`,
				routerPropsString,
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
						fmt.Sprintf(`<script type="application/javascript">function target(){return document.getElementById("%s")}</script>`, targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf(`<div id="%s"></div>`, targetId),
					1,
				),
				"<!--app-head-->",
				"",
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				`<script type="application/javascript">function props(){return %s}</script>`,
				routerPropsString,
			),
			1,
		)
	} else if ModeServer == configuration.Render {
		head, body, renderError := render(response, routerPropsString, configuration.Globals)
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
					fmt.Sprintf(`<div id="%s">%s</div>`, targetId, body),
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
