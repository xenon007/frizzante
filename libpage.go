package frizzante

import (
	"embed"
	"encoding/json"
	"fmt"
	uuid "github.com/nu7hatch/gouuid"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Page struct {
	renderMode         RenderMode
	data               map[string]any
	embeddedFileSystem embed.FS
	pageId             string
	path               map[string]string
}

func PageWithRenderMode(self *Page, renderMode RenderMode) {
	self.renderMode = renderMode
}

func PageWithData(self *Page, key string, value any) {
	self.data[key] = value
}

var noScriptPattern = regexp.MustCompile(`<script.*>.*</script>`)
var pagesToPaths = map[string]string{}

type svelteRouterProps struct {
	PageId string            `json:"pageId"`
	Data   map[string]any    `json:"data"`
	Paths  map[string]string `json:"paths"`
	Path   map[string]string `json:"path"`
}

// PageCompile compiles a svelte page.
func PageCompile(self *Page) (string, error) {
	if nil == self {
		self = &Page{
			renderMode: RenderModeFull,
			data:       map[string]any{},
			path:       map[string]string{},
		}
	} else {
		if nil == self.data {
			self.data = map[string]any{}
		}
	}

	fileNameIndex := filepath.Join(".dist", "client", ".frizzante", "vite-project", "index.html")

	var indexBytes []byte

	if "1" == os.Getenv("DEV") {
		indexBytesLocal, readError := os.ReadFile(fileNameIndex)
		if readError != nil {
			return "", readError
		}
		indexBytes = indexBytesLocal
	} else {
		indexBytesLocal, readError := self.embeddedFileSystem.ReadFile(fileNameIndex)
		if readError != nil {
			return "", readError
		}
		indexBytes = indexBytesLocal
	}

	routerPropsBytes, jsonError := json.Marshal(svelteRouterProps{
		PageId: self.pageId,
		Data:   self.data,
		Paths:  pagesToPaths,
		Path:   self.path,
	})

	if jsonError != nil {
		return "", jsonError
	}

	routerPropsString := string(routerPropsBytes)

	targetId, targetIdError := uuid.NewV4()
	if targetIdError != nil {
		return "", targetIdError
	}

	if RenderModeFull == self.renderMode {
		head, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		return strings.Replace(
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
		), nil
	}

	if RenderModeClient == self.renderMode {
		return strings.Replace(
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
		), nil
	}

	if RenderModeServer == self.renderMode {
		head, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		return strings.Replace(
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
		), nil
	}

	if RenderModeHeadless == self.renderMode {
		_, body, renderError := render(self.embeddedFileSystem, routerPropsString)
		if renderError != nil {
			return "", renderError
		}

		body = strings.Replace(body, "<!--[-->", "", -1)
		body = strings.Replace(body, "<!--]-->", "", -1)
		body = strings.Replace(body, "<!--!]-->", "", -1)
		body = strings.Replace(body, "<!--[!-->", "", -1)
		body = strings.Replace(body, "<!---->", "", -1)

		return body, nil

	}

	return "", nil
}

// PageCreate creates a page.
func PageCreate(
	embeddedFileSystem embed.FS,
	renderMode RenderMode,
	pageId string,
	data map[string]any,
) *Page {
	if nil == data {
		data = map[string]any{}
	}

	return &Page{
		renderMode:         renderMode,
		data:               data,
		pageId:             pageId,
		embeddedFileSystem: embeddedFileSystem,
		path:               map[string]string{},
	}
}
