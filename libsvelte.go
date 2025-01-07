package frizzante

import (
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"path/filepath"
	"rogchap.com/v8go"
	"strings"
)

// EchoSvelte renders and echos the svelte application.
func EchoSvelte(response *Response, props map[string]interface{}) {
	renderFileName := filepath.Join(response.server.wwwDirectory, "dist", "server", "render.server.js")

	renderEsmBytes, readError := response.server.embeddedFileSystem.ReadFile(renderFileName)
	if readError != nil {
		ServerNotifyError(response.server, readError)
		return
	}

	renderEsm := string(renderEsmBytes)

	renderCjs, javaScriptBundleError := JavaScriptBundle(response.server.wwwDirectory, api.FormatCommonJS, renderEsm)
	if javaScriptBundleError != nil {
		ServerNotifyError(response.server, javaScriptBundleError)
		return
	}

	renderIif := fmt.Sprintf("const module={exports:{}}; const render = \n(function(){\n%s\nreturn render;\n})()", renderCjs)

	bytesProps, jsonError := json.Marshal(props)
	if jsonError != nil {
		ServerNotifyError(response.server, jsonError)
		return
	}
	stringProps := string(bytesProps)

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

	doneCjs, bundleError := JavaScriptBundle(response.server.wwwDirectory, api.FormatCommonJS, doneEsm)
	if bundleError != nil {
		ServerNotifyError(response.server, bundleError)
		return
	}

	head := ""
	body := ""
	_, destroy, javaScriptError := JavaScriptRun(
		doneCjs, map[string]v8go.FunctionCallback{
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
		},
	)
	if javaScriptError != nil {
		ServerNotifyError(response.server, javaScriptError)
		return
	}
	defer destroy()

	indexBytes, readError := response.server.embeddedFileSystem.ReadFile(filepath.Join(response.server.wwwDirectory, "dist", "client", "index.html"))
	if readError != nil {
		return
	}

	scriptTarget := fmt.Sprintf(`<script type="application/javascript">function target(){return document.getElementById("app")}</script>`)
	scriptProps := fmt.Sprintf(`<script type="application/javascript">function props(){return %s}</script>`, stringProps)
	scriptBody := fmt.Sprintf(`<div id="app">%s</div>`, body)

	index := strings.Replace(string(indexBytes), "<!--app-head-->", head, 1)
	index = strings.Replace(index, "<!--app-target-->", scriptTarget, 1)
	index = strings.Replace(index, "<!--app-props-->", scriptProps, 1)
	index = strings.Replace(index, "<!--app-body-->", scriptBody, 1)

	Header(response, "Content-Type", "text/html")
	Echo(response, index)
}
