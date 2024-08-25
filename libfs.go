package frizzante

import (
	"errors"
	"os"
)

func exists(fileName string) bool {
	_, statError := os.Stat(fileName)
	return nil == statError || !errors.Is(statError, os.ErrNotExist)
}

func existsAndIsFile(fileName string) bool {
	stat, statError := os.Stat(fileName)
	existsLocal := nil == statError || !errors.Is(statError, os.ErrNotExist)
	return existsLocal && !stat.IsDir()
}

func existsAndIsDirectory(fileName string) bool {
	stat, statError := os.Stat(fileName)
	existsLocal := nil == statError || !errors.Is(statError, os.ErrNotExist)
	return existsLocal && stat.IsDir()
}
