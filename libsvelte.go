package frizzante

import (
	"encoding/json"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"path/filepath"
	"rogchap.com/v8go"
	"strings"
)

func render(response *Response, stringProps string) (string, string, error) {
	renderFileName := filepath.Join(response.server.wwwDirectory, "dist", "server", "render.server.js")

	renderEsmBytes, readError := response.server.embeddedFileSystem.ReadFile(renderFileName)
	if readError != nil {
		return "", "", readError
	}

	renderEsm := string(renderEsmBytes)

	renderCjs, javaScriptBundleError := JavaScriptBundle(response.server.wwwDirectory, api.FormatCommonJS, renderEsm)
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

	doneCjs, bundleError := JavaScriptBundle(response.server.wwwDirectory, api.FormatCommonJS, doneEsm)
	if bundleError != nil {
		return "", "", bundleError
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
		return head, body, javaScriptError
	}
	defer destroy()

	return head, body, nil
}

// Svelte renders and echos the svelte application.
func Svelte(response *Response, props map[string]interface{}) {
	indexBytes, readError := response.server.embeddedFileSystem.ReadFile(filepath.Join(response.server.wwwDirectory, "dist", "client", "index.html"))
	if readError != nil {
		return
	}

	ssr := props["ssr"].(bool)

	bytesProps, jsonError := json.Marshal(props)
	if jsonError != nil {
		ServerNotifyError(response.server, jsonError)
		return
	}
	stringProps := string(bytesProps)

	head := ""
	body := ""
	if ssr {
		headLocal, bodyLocal, renderError := render(response, stringProps)
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
