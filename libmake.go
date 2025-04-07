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

func createIndex(indexName string) {
	if "" == indexName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the index: ")
		indexName, _ = reader.ReadString('\n')
		if "" == indexName {
			createIndex(indexName)
			return
		}
	}

	indexName = strings.ReplaceAll(indexName, "-", "_")

	indexNameCamel := strings.Trim(strings.ToLower(indexName[0:1])+indexName[1:], "\r\n\t ")
	indexNamePascal := strings.Trim(strings.ToTitle(indexName[0:1])+indexName[1:], "\r\n\t ")

	oldFileName := "templates/indexes/example.go"
	newFileName := filepath.Join("lib", "indexes", indexNameCamel+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}
	if Exists(newFileName) {
		fmt.Printf("Index `%s` already exists.", indexNameCamel)
		return
	}

	// Index.
	oldTitle := []byte("func Index")
	newTitle := []byte("func " + indexNamePascal)
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	// Show.
	oldTitle = []byte("indexShow")
	newTitle = []byte(indexNameCamel + "Show")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	// Action.
	oldTitle = []byte("indexAction")
	newTitle = []byte(indexNameCamel + "Action")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}

}

func createGuard(guardName string) {
	if "" == guardName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the guard: ")
		guardName, _ = reader.ReadString('\n')
		if "" == guardName {
			createIndex(guardName)
			return
		}
	}

	guardName = strings.ReplaceAll(guardName, "-", "_")

	guardNameCamel := strings.Trim(strings.ToLower(guardName[0:1])+guardName[1:], "\r\n\t ")
	guardNamePascal := strings.Trim(strings.ToTitle(guardName[0:1])+guardName[1:], "\r\n\t ")

	oldFileName := "templates/guards/example.go"
	newFileName := filepath.Join("lib", "guards", guardNameCamel+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}

	if Exists(newFileName) {
		fmt.Printf("Guard `%s` already exists.", guardNameCamel)
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
	if "" == pageName {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name the page: ")
		pageName, _ = reader.ReadString('\n')
		if "" == pageName {
			createIndex(pageName)
			return
		}
	}

	pageName = strings.ReplaceAll(pageName, "-", "_")

	//pageNameCamel := strings.Trim(strings.ToLower(pageName[0:1])+pageName[1:], "\r\n\t ")
	pageNamePascal := strings.Trim(strings.ToTitle(pageName[0:1])+pageName[1:], "\r\n\t ")

	oldFileName := "templates/pages/example.svelte"
	newFileName := filepath.Join("lib", "pages", pageNamePascal+".svelte")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}

	if Exists(newFileName) {
		fmt.Printf("Page `%s` already exists.", pageNamePascal)
		return
	}

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}
}

// Make makes things.
func Make() {
	index := flag.Bool("index", false, "")
	guard := flag.Bool("guard", false, "")
	page := flag.Bool("page", false, "")
	name := flag.String("name", "", "")
	flag.Parse()

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
