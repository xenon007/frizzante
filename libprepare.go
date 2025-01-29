package frizzante

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed vite-project/*
var svelteRenderToolsFileSystem embed.FS

var sveltePagesToFileNames = map[string]string{}

// PrepareSveltePages prepares a directory of svelte page.
func PrepareSveltePages(directoryName string) {
	err := filepath.Walk(directoryName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(info.Name(), ".svelte") {
				return nil
			}

			fileName := info.Name()
			pageId := strings.ReplaceAll(
				strings.ReplaceAll(
					strings.TrimSuffix(fileName, ".svelte"),
					"/",
					"::",
				),
				`\`,
				"::",
			)

			PrepareSveltePage(pageId, filepath.Join(directoryName, fileName))

			return nil
		},
	)
	if err != nil {

		panic(err)

	}
}

// PrepareSveltePage prepares a svelte page.
func PrepareSveltePage(pageId string, fileName string) {
	relativeFileName, err := filepath.Rel("www/.frizzante/vite-project", fileName)
	if err != nil {
		panic(err)
	}
	sveltePagesToFileNames[pageId] = fmt.Sprintf("./%s", relativeFileName)
}

// PrepareStart begins preparation.
func PrepareStart() {
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

// PrepareEnd ends preparation by generating all prepared code.
func PrepareEnd() {
	dumpSvelteFiles()
}

func dumpSvelteFiles() {
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
	builder.WriteString("    let {pageId, paths, data} = $props()\n")
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
