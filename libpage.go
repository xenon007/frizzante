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

var pages = map[string]string{}

type Page struct {
	render     Render
	data       map[string]any
	efs        embed.FS
	name       string
	parameters map[string]string
}

// PageWithRender sets the page rendering mode.
func PageWithRender(self *Page, render Render) {
	self.render = render
}

// PageWithData sets data to the page.
func PageWithData(self *Page, key string, value any) {
	self.data[key] = value
}

var noScriptPattern = regexp.MustCompile(`<script.*>.*</script>`)

type PageProps struct {
	Page       string            `json:"page"`
	Data       map[string]any    `json:"data"`
	Pages      map[string]string `json:"pages"`
	Parameters map[string]string `json:"parameters"`
}

// PageCompile compiles a page.
func PageCompile(self *Page) (string, error) {
	if nil == self.parameters {
		self.parameters = map[string]string{}
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
		indexBytesLocal, readError := self.efs.ReadFile(fileNameIndex)
		if readError != nil {
			return "", readError
		}
		indexBytes = indexBytesLocal
	}

	routerPropsBytes, jsonError := json.Marshal(PageProps{
		Pages:      pages,
		Page:       self.name,
		Data:       self.data,
		Parameters: self.parameters,
	})

	if jsonError != nil {
		return "", jsonError
	}

	routerPropsString := string(routerPropsBytes)

	targetId, targetIdError := uuid.NewV4()
	if targetIdError != nil {
		return "", targetIdError
	}

	if RenderFull == self.render {
		head, body, renderError := render(self.efs, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		return strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						fmt.Sprintf("<script type=\"application/javascript\">function target(){return document.getElementById(\"%s\")}</script>", targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf("<div id=\"%s\">%s</div>", targetId, body),
					1,
				),
				"<!--app-head-->",
				head,
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				"<script type=\"application/javascript\">function props(){return %s}</script>",
				routerPropsString,
			),
			1,
		), nil
	}

	if RenderClient == self.render {
		return strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						string(indexBytes),
						"<!--app-target-->",
						fmt.Sprintf("<script type=\"application/javascript\">function target(){return document.getElementById(\"%s\")}</script>", targetId),
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf("<div id=\"%s\"></div>", targetId),
					1,
				),
				"<!--app-head-->",
				"",
				1,
			),
			"<!--app-data-->",
			fmt.Sprintf(
				"<script type=\"application/javascript\">function props(){return %s}</script>",
				routerPropsString,
			),
			1,
		), nil
	}

	if RenderServer == self.render {
		head, body, renderError := render(self.efs, routerPropsString)
		if renderError != nil {
			return "", renderError
		}
		return strings.Replace(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						noScriptPattern.ReplaceAllString(string(indexBytes), ""),
						"<!--app-target-->",
						"",
						1,
					),
					"<!--app-body-->",
					fmt.Sprintf("<div id=\"%s\">%s</div>", targetId, body),
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

	if RenderHeadless == self.render {
		_, body, renderError := render(self.efs, routerPropsString)
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
