package frizzante

import (
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"os"
	"path/filepath"
	//"os"
	"rogchap.com/v8go"
)

func structuredClone(info *v8go.FunctionCallbackInfo) *v8go.Value {
	args := info.Args()
	if len(args) > 0 {
		contextLocal := info.Context()
		value := args[0]

		stringified, stringifyError := v8go.JSONStringify(contextLocal, value)
		if stringifyError != nil {
			return nil
		}

		parsed, parseError := v8go.JSONParse(contextLocal, stringified)
		if parseError != nil {
			return nil
		}

		return parsed
	}

	return nil
}

type JavaScriptContext struct {
	isolate *v8go.Isolate
	global  *v8go.ObjectTemplate
	context *v8go.Context
}

func newJavaScriptContext(globals map[string]v8go.FunctionCallback) (*JavaScriptContext, error) {
	isolate := v8go.NewIsolate()
	global := v8go.NewObjectTemplate(isolate)

	for key, callback := range globals {
		template := v8go.NewFunctionTemplate(isolate, callback)
		setError := global.Set(key, template)
		if setError != nil {
			return nil, setError
		}
	}

	context := v8go.NewContext(isolate, global)

	return &JavaScriptContext{
		isolate: isolate,
		global:  global,
		context: context,
	}, nil
}

// JavaScript runs a javascript module.
//
// It returns the last expression of the script and a destroyer function.
//
// You should always call the destroyer function as soon as possible to limit memory usage.
//
// Each global function will be injected into the context of the module automatically so that you can invoke them from the script.
func JavaScript(source string, globals map[string]v8go.FunctionCallback) (*v8go.Value, func(), error) {
	js, createError := newJavaScriptContext(globals)
	if createError != nil {
		return nil, nil, createError
	}
	exports, runError := js.context.RunScript(source, "frizzante.js")
	if runError != nil {
		return nil, nil, runError
	}

	return exports, func() { JavaScriptDestroy(js) }, nil
}

func JavaScriptDestroy(js *JavaScriptContext) {
	js.context.Close()
	js.isolate.Dispose()
}

func componentsPlugin(server *Server, id string) api.Plugin {
	return api.Plugin{
		Name: "svelte",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(
				api.OnResolveOptions{Filter: `^\$components\/`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					path := filepath.Join(args.ResolveDir, args.Path)

					return api.OnResolveResult{
						Path:      path,
						Namespace: "svelte-ns",
					}, nil
				},
			)

			build.OnLoad(
				api.OnLoadOptions{Filter: `.*`, Namespace: "svelte-ns"},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, readError := os.ReadFile(args.Path)
					if readError != nil {
						return api.OnLoadResult{}, readError
					}

					js, _ := SvelteCompile(server, args.Path, false, string(contents))

					return api.OnLoadResult{
						Contents: &js,
						Loader:   api.LoaderJS,
					}, nil
				},
			)
		},
	}
}

func scriptsPlugin(server *Server, id string) api.Plugin {
	return api.Plugin{
		Name: "scripts",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(
				api.OnResolveOptions{Filter: `^\$scripts\/`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					path := filepath.Join(args.ResolveDir, args.Path)

					return api.OnResolveResult{
						Path:      path,
						Namespace: "scripts-ns",
					}, nil
				},
			)

			build.OnLoad(
				api.OnLoadOptions{Filter: `.*`, Namespace: "scripts-ns"},
				func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					contents, readError := os.ReadFile(args.Path)
					if readError != nil {
						return api.OnLoadResult{}, readError
					}

					js := string(contents)

					return api.OnLoadResult{
						Contents: &js,
						Loader:   api.LoaderJS,
					}, nil
				},
			)
		},
	}
}

func Bundle(server *Server, id string, source string) (string, error) {
	result := api.Build(api.BuildOptions{
		Bundle: true,
		Format: api.FormatESModule,
		Write:  false,
		Stdin: &api.StdinOptions{
			Contents:   source,
			ResolveDir: server.uiDirectory,
		},
		Loader: map[string]api.Loader{
			".svelte": api.LoaderText,
		},
		Plugins: []api.Plugin{
			componentsPlugin(server, id),
			scriptsPlugin(server, id),
		},
	})

	for _, err := range result.Errors {
		return "", fmt.Errorf("%s in %s:%d:%d", err.Text, err.Location.File, err.Location.Line, err.Location.Column)
	}

	return string(result.OutputFiles[0].Contents), nil
}
