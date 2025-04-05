package frizzante

import (
	"embed"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"os"
	"path/filepath"
	"rogchap.com/v8go"
)

type Render int64

const (
	RenderServer   Render = 0 // Renders only on the server.
	RenderClient   Render = 1 // Renders only on the client.
	RenderFull     Render = 2 // Renders on both the server and the client.
	RenderHeadless Render = 3 // Renders only on the server and omits the base template.
)

func render(efs embed.FS, stringProps string) (string, string, error) {
	renderFileName := filepath.Join(".dist", "server", "render.server.js")

	var renderEsmBytes []byte
	if "1" == os.Getenv("DEV") {
		renderEsmBytesLocal, readError := os.ReadFile(renderFileName)
		if readError != nil {
			return "", "", readError
		}
		renderEsmBytes = renderEsmBytesLocal
	} else {
		renderEsmBytesLocal, readError := efs.ReadFile(renderFileName)
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
	globals := map[string]v8go.FunctionCallback{
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

	_, destroy, javaScriptError := JavaScriptRun(doneCjs, globals)
	if javaScriptError != nil {
		return head, body, javaScriptError
	}
	defer destroy()

	return head, body, nil
}
