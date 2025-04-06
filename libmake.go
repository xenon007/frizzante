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

	indexName = strings.Trim(strings.ToTitle(indexName[0:1])+indexName[1:], "\r\n\t ")

	oldFileName := "templates/indexes/example.go"
	newFileName := filepath.Join("lib", "indexes", strings.ToLower(indexName)+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}
	if Exists(newFileName) {
		fmt.Printf("Index `%s` already exists.", indexName)
		return
	}

	oldTitle := []byte("func index")
	newTitle := []byte("func " + indexName)
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

	guardName = strings.Trim(strings.ToTitle(guardName[0:1])+guardName[1:], "\r\n\t ")

	oldFileName := "templates/guards/example.go"
	newFileName := filepath.Join("lib", "guards", strings.ToLower(guardName)+".go")
	readBytes, readError := templates.ReadFile(oldFileName)
	if nil != readError {
		panic(readError)
	}

	if Exists(newFileName) {
		fmt.Printf("Guard `%s` already exists.", guardName)
		return
	}

	// Api.
	oldTitle := []byte("func guardApi")
	newTitle := []byte("func " + guardName + "Api")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	// Pages.
	oldTitle = []byte("func guardPages")
	newTitle = []byte("func " + guardName + "Pages")
	readBytes = bytes.ReplaceAll(readBytes, oldTitle, newTitle)

	writeError := os.WriteFile(newFileName, readBytes, os.ModePerm)
	if writeError != nil {
		panic(writeError)
	}
}

func Make() {
	index := flag.Bool("index", false, "")
	guard := flag.Bool("guard", false, "")
	name := flag.String("name", "", "")
	flag.Parse()

	if *index {
		createIndex(*name)
	}

	if *guard {
		createGuard(*name)
	}
}
