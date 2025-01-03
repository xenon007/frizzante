package frizzante

import (
	"rogchap.com/v8go"
	"strings"
)

func Svelte(response *Response, id string, source string) {
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
			const payload = { out: '' }
			_unknown_(payload)
			output(payload.out)
			`

		thirdStepScript, thirdStepScriptError := Bundle(secondStepScriptFixed)
		if thirdStepScriptError != nil {
			ServerNotifyError(server, thirdStepScriptError)
			return
		}

		js = thirdStepScript

		setTempJsError := ServerSetTemporaryFile(server, jsBundleName, thirdStepScript)
		if setTempJsError != nil {
			ServerNotifyError(server, setTempJsError)
			return
		}

		setTempCssError := ServerSetTemporaryFile(server, cssBundleName, css)
		if setTempCssError != nil {
			ServerNotifyError(server, setTempCssError)
			return
		}
	} else {
		thirdStepScript, getTempJsError := ServerGetTemporaryFile(server, jsBundleName)
		if getTempJsError != nil {
			ServerNotifyError(server, getTempJsError)
			return
		}
		js = thirdStepScript

		cssLocal, getTempCssError := ServerGetTemporaryFile(server, cssBundleName)
		if getTempCssError != nil {
			ServerNotifyError(server, getTempCssError)
			return
		}
		css = cssLocal
	}

	html := ""
	_, destroyRender, renderError := JavaScript(js, map[string]v8go.FunctionCallback{
		"output": func(info *v8go.FunctionCallbackInfo) *v8go.Value {
			args := info.Args()
			if len(args) > 0 {
				html = args[0].String()
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
	Echo(response, html)
}
