package frizzante

import (
	"fmt"
	"rogchap.com/v8go"
	"strings"
)

// Svelte renders and echos svelte code.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func Svelte(response *Response, id string, source string) {
	if len(id) > 255 {
		ServerNotifyError(response.server, fmt.Errorf("page id is too long"))
		return
	}

	server := response.server
	js := ""
	css := ""

	jsBundleName := id
	if !strings.HasSuffix(id, "/") {
		jsBundleName += "/"
	}
	jsBundleName += "bundle.js"

	cssBundleName := id
	if !strings.HasSuffix(id, "/") {
		cssBundleName += "/"
	}
	cssBundleName += "bundle.css"

	if !ServerHasTemporaryFile(server, jsBundleName) ||
		!ServerHasTemporaryFile(server, cssBundleName) {
		secondStepScript := ""
		firstStepScript, firstStepScriptError := Bundle(`
				import { compile } from 'svelte/compiler'
				const result = compile(source(),{ generate: generate() })
				js(result.js?.code??'')
				css(result.css?.code??'')
			`)
		if firstStepScriptError != nil {
			ServerNotifyError(server, firstStepScriptError)
			return
		}

		_, destroy, runError := JavaScript(firstStepScript, map[string]v8go.FunctionCallback{
			"structuredClone": structuredClone,
			"source": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
				value, valueError := v8go.NewValue(info.Context().Isolate(), source)
				if valueError != nil {
					ServerNotifyError(server, valueError)
					return nil
				}
				return value
			},
			"generate": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
				value, valueError := v8go.NewValue(info.Context().Isolate(), "server")
				if valueError != nil {
					ServerNotifyError(server, valueError)
					return nil
				}
				return value
			},
			"js": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
				if len(info.Args()) > 0 {
					secondStepScript = info.Args()[0].String()
				}
				return nil
			},
			"css": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
				if len(info.Args()) > 0 {
					css = info.Args()[0].String()
				}
				return nil
			},
		})
		if runError != nil {
			ServerNotifyError(server, runError)
			return
		}

		defer destroy()

		secondStepScriptFixed := strings.Replace(secondStepScript, "export default function", "function", 1)
		secondStepScriptFixed += `
			const payload = { head: '', out: '', body: '' }
			_unknown_(payload)
			head(payload.head)
			out(payload.out)
			`

		thirdStepScript, thirdStepScriptError := Bundle(secondStepScriptFixed)
		if thirdStepScriptError != nil {
			ServerNotifyError(server, thirdStepScriptError)
			return
		}

		js = thirdStepScript

		ServerSetTemporaryFile(server, jsBundleName, thirdStepScript)
		ServerSetTemporaryFile(server, cssBundleName, css)
	} else {
		js = ServerGetTemporaryFile(server, jsBundleName)
		css = ServerGetTemporaryFile(server, cssBundleName)
	}

	head := ""
	out := ""
	_, destroyRender, renderError := JavaScript(js, map[string]v8go.FunctionCallback{
		"inspect": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			if len(info.Args()) > 0 {
				println(info.Args()[0].String())
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
		"out": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				out = args[0].String()
			}
			return nil
		},
	})
	if renderError != nil {
		ServerNotifyError(server, renderError)
		return
	}
	defer destroyRender()

	Header(response, "Content-Type", "text/html")
	html := strings.Replace(
		strings.Replace(
			"<!doctype html><html lang=\"en\"><head>%head%</head><body>%out%</body></html>",
			"%out%",
			out,
			-1,
		),
		"%head%",
		head,
		-1,
	)
	Echo(response, html)
}

// SveltePage renders anc echos a svelte page.
//
// When id is longer than 255 characters, the operation will fail silently and the server will be notified.
func SveltePage(response *Response, id string) {
	Svelte(response, id, ServerGetPage(response.server, id))
}
