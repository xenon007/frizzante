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
	relativeFileName, err := filepath.Rel(".frizzante/vite-project", fileName)
	if err != nil {
		panic(err)
	}
	sveltePagesToFileNames[pageId] = fmt.Sprintf("./%s", relativeFileName)
}

// PrepareStart begins preparation.
func PrepareStart() {
	err := prepareSveltePagesStart()
	if err != nil {
		panic(err)
	}
}

// PrepareEnd ends preparation by generating all prepared code.
func PrepareEnd() {
	err := prepareSveltePagesEnd()
	if err != nil {
		panic(err)
	}
}

func prepareServerLoader() error {
	var builder strings.Builder
	renderServerSvelte, readError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.server.svelte")
	if readError != nil {
		return readError
	}
	for id, fileName := range sveltePagesToFileNames {
		builder.WriteString(fmt.Sprintf("    import %s from '%s'\n", strings.ToUpper(id), fileName))
	}

	renderServerSvelteString := strings.Replace(string(renderServerSvelte), "//:app-imports", builder.String(), 1)

	builder.Reset()
	counter := 0
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

	renderServerSvelteString = strings.Replace(renderServerSvelteString, "<!--app-router-->", builder.String(), 1)

	writeError := os.WriteFile(".frizzante/vite-project/render.server.svelte", []byte(renderServerSvelteString), os.ModePerm)
	if writeError != nil {
		return writeError
	}
	return nil
}

func prepareClientLoader() error {
	// Build client loader.
	renderClientSvelte, readError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.client.svelte")
	if readError != nil {
		return readError
	}

	var builder strings.Builder
	builder.WriteString("import Page from './page.async.svelte'")
	renderClientSvelteString := strings.Replace(string(renderClientSvelte), "//:app-imports", builder.String(), 1)

	builder.Reset()
	counter := 0
	for pageId, fileName := range sveltePagesToFileNames {
		if 0 == counter {
			builder.WriteString(fmt.Sprintf("{#if '%s' === pageIdState}\n", pageId))
		} else {
			builder.WriteString(fmt.Sprintf("{:else if '%s' === pageIdState}\n", pageId))
		}
		builder.WriteString(fmt.Sprintf("    <Page from={import('%s')} />\n", fileName))
		counter++
	}
	if counter > 0 {
		builder.WriteString("{/if}")
	}
	renderClientSvelteString = strings.Replace(renderClientSvelteString, "<!--app-router-->", builder.String(), 1)

	// Dump client loader.
	writeError := os.WriteFile(".frizzante/vite-project/render.client.svelte", []byte(renderClientSvelteString), os.ModePerm)
	if writeError != nil {
		return writeError
	}

	return nil
}

func prepareSveltePagesStart() error {
	asyncSvelte, asyncSvelteError := svelteRenderToolsFileSystem.ReadFile("vite-project/page.async.svelte")
	if asyncSvelteError != nil {
		return asyncSvelteError
	}

	indexHtml, indexHtmlError := svelteRenderToolsFileSystem.ReadFile("vite-project/index.html")
	if indexHtmlError != nil {
		return indexHtmlError
	}

	renderClientJs, renderClientJsError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.client.js")
	if renderClientJsError != nil {
		return renderClientJsError
	}

	renderServerJs, renderServerJsError := svelteRenderToolsFileSystem.ReadFile("vite-project/render.server.js")
	if renderServerJsError != nil {
		return renderServerJsError
	}

	submitSvelte, submitSvelteError := svelteRenderToolsFileSystem.ReadFile("vite-project/lib/components/Submit.svelte")
	if submitSvelteError != nil {
		return submitSvelteError
	}

	if !Exists(".frizzante/vite-project") {
		err := os.MkdirAll(".frizzante/vite-project", os.ModePerm)
		if err != nil {
			return err
		}
	}

	err := os.WriteFile(".frizzante/vite-project/page.async.svelte", asyncSvelte, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(".frizzante/vite-project/index.html", indexHtml, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(".frizzante/vite-project/render.client.js", renderClientJs, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(".frizzante/vite-project/render.server.js", renderServerJs, os.ModePerm)
	if err != nil {
		return err
	}

	if !Exists(".frizzante/vite-project/lib/components") {
		err = os.MkdirAll(".frizzante/vite-project/lib/components", os.ModePerm)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(".frizzante/vite-project/lib/components/Submit.svelte", submitSvelte, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func prepareSveltePagesEnd() error {
	if !Exists(".frizzante/vite-project") {
		err := os.MkdirAll(".frizzante/vite-project", os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Prepare server loader.
	dumpServerLoaderError := prepareServerLoader()
	if dumpServerLoaderError != nil {
		return dumpServerLoaderError
	}

	// Prepare client loader.
	dumpClientLoaderError := prepareClientLoader()
	if dumpClientLoaderError != nil {
		return dumpServerLoaderError
	}
	return nil
}
