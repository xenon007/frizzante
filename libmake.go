package frizzante

import (
	"bufio"
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*/**
var templates embed.FS

func createApi(apiName string) {
	fileName := filepath.Join("lib", "api")
	if !Exists(fileName) {
		writeError := os.MkdirAll(fileName, os.ModePerm)
		if writeError != nil {
			panic(writeError)
		}
	}

	if "" == apiName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the api: ")
		apiName, _ = reader.ReadString('\n')
		if "" == apiName {
			createApi(apiName)
			return
		}
	}

	apiName = strings.Trim(strings.ReplaceAll(apiName, "-", "_"), "\r\n\t ")

	apiNameCamel := strings.ToLower(apiName[0:1]) + apiName[1:]
	apiNamePascal := strings.ToTitle(apiName[0:1]) + apiName[1:]

	oldFileName := "templates/api/example.go"
	newFileName := filepath.Join("lib", "api", apiNameCamel+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}
	if Exists(newFileName) {
		fmt.Printf("Api `%s` already exists.\n", apiNameCamel)
		return
	}

	// Api.
	oldName := []byte("func Api")
	newName := []byte("func " + apiNamePascal)
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Pattern.
	oldName = []byte("\"GET /\"")
	newName = []byte("\"GET /api/" + apiNameCamel + "\"")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Serve.
	oldName = []byte("serveFunction")
	newName = []byte(apiNameCamel + "ServeFunction")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}

}

func createIndex(indexName string) {
	fileName := filepath.Join("lib", "indexes")
	if !Exists(fileName) {
		writeError := os.MkdirAll(fileName, os.ModePerm)
		if writeError != nil {
			panic(writeError)
		}
	}

	if "" == indexName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the index: ")
		indexName, _ = reader.ReadString('\n')
		if "" == indexName {
			createIndex(indexName)
			return
		}
	}

	indexName = strings.Trim(strings.ReplaceAll(indexName, "-", "_"), "\r\n\t ")

	indexNameCamel := strings.ToLower(indexName[0:1]) + indexName[1:]
	indexNamePascal := strings.ToTitle(indexName[0:1]) + indexName[1:]

	oldFileName := "templates/indexes/example.go"
	newFileName := filepath.Join("lib", "indexes", indexNameCamel+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}
	if Exists(newFileName) {
		fmt.Printf("Index `%s` already exists.\n", indexNameCamel)
		return
	}

	// Index.
	oldName := []byte("func Index")
	newName := []byte("func " + indexNamePascal)
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Path.
	oldName = []byte("\"/path\"")
	newName = []byte("\"/" + indexName + "\"")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Page.
	oldName = []byte("\"page\"")
	newName = []byte("\"" + indexName + "\"")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Show.
	oldName = []byte("indexShowFunction")
	newName = []byte(indexNameCamel + "ShowFunction")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	// Action.
	oldName = []byte("indexActionFunction")
	newName = []byte(indexNameCamel + "ActionFunction")
	readBytes = bytes.ReplaceAll(readBytes, oldName, newName)

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}

}

func createGuard(guardName string) {
	fileName := filepath.Join("lib", "guards")
	if !Exists(fileName) {
		writeError := os.MkdirAll(fileName, os.ModePerm)
		if writeError != nil {
			panic(writeError)
		}
	}

	if "" == guardName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the guard: ")
		guardName, _ = reader.ReadString('\n')
		if "" == guardName {
			createGuard(guardName)
			return
		}
	}

	guardName = strings.Trim(strings.ReplaceAll(guardName, "-", "_"), "\r\n\t ")

	guardNameCamel := strings.ToLower(guardName[0:1]) + guardName[1:]
	guardNamePascal := strings.ToTitle(guardName[0:1]) + guardName[1:]

	oldFileName := "templates/guards/example.go"
	newFileName := filepath.Join("lib", "guards", guardNameCamel+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}

	if Exists(newFileName) {
		fmt.Printf("Guard `%s` already exists.\n", guardNameCamel)
		return
	}

	// Api.
	oldTitle := []byte("GuardApi")
	newTitle := []byte(guardNamePascal + "Api")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	// Pages.
	oldTitle = []byte("GuardPages")
	newTitle = []byte(guardNamePascal + "Pages")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}
}

func createPage(pageName string) {
	fileName := filepath.Join("lib", "pages")
	if !Exists(fileName) {
		writeError := os.MkdirAll(fileName, os.ModePerm)
		if writeError != nil {
			panic(writeError)
		}
	}

	if "" == pageName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the page: ")
		pageName, _ = reader.ReadString('\n')
		if "" == pageName {
			createPage(pageName)
			return
		}
	}

	pageName = strings.Trim(strings.ReplaceAll(pageName, "-", "_"), "\r\n\t ")

	//pageNameCamel := strings.ToLower(pageName[0:1])+pageName[1:]
	//pageNamePascal := strings.ToTitle(pageName[0:1]) + pageName[1:]

	oldFileName := "templates/pages/example.svelte"
	newFileName := filepath.Join("lib", "pages", pageName+".svelte")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}

	if Exists(newFileName) {
		fmt.Printf("Page `%s` already exists.\n", pageName)
		return
	}

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}
}

// Make makes things.
func Make() {
	api := flag.Bool("api", false, "")
	index := flag.Bool("index", false, "")
	guard := flag.Bool("guard", false, "")
	page := flag.Bool("page", false, "")
	name := flag.String("name", "", "")
	flag.Parse()

	if *api {
		createApi(*name)
	}

	if *index {
		createIndex(*name)
	}

	if *guard {
		createGuard(*name)
	}

	if *page {
		createPage(*name)
	}
}
