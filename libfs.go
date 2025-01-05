package frizzante

import (
	"embed"
	"errors"
	"os"
	"strings"
)

// EmbeddedExists checks if fileName exists.
func EmbeddedExists(embeddedFileSystem embed.FS, fileName string) bool {
	return EmbeddedIsFile(embeddedFileSystem, fileName) || EmbeddedIsDirectory(embeddedFileSystem, fileName)
}

// EmbeddedIsFile check if fileName exists and is a file.
func EmbeddedIsFile(embeddedFileSystem embed.FS, fileName string) bool {
	_, err := embeddedFileSystem.ReadFile(fileName)
	if err != nil {
		return false
	}
	return true
}

// EmbeddedIsDirectory checks if fileName exists and is a directory.
func EmbeddedIsDirectory(embeddedFileSystem embed.FS, fileName string) bool {
	_, err := embeddedFileSystem.ReadDir(fileName)
	if err != nil {
		return false
	}
	return true
}

// Exists checks if fileName exists.
func Exists(fileName string) bool {
	_, statError := os.Stat(fileName)
	return nil == statError || !errors.Is(statError, os.ErrNotExist)
}

// IsFile check if fileName exists and is a file.
func IsFile(fileName string) bool {
	stat, statError := os.Stat(fileName)
	if statError != nil {
		return !errors.Is(statError, os.ErrNotExist)
	}
	return !stat.IsDir()
}

// IsDirectory checks if fileName exists and is a directory.
func IsDirectory(fileName string) bool {
	stat, statError := os.Stat(fileName)
	if statError != nil {
		return !errors.Is(statError, os.ErrNotExist)
	}
	return stat.IsDir()
}

func Mime(fileName string) string {
	mime := "text/plain"

	if strings.HasSuffix(fileName, ".html") {
		mime = "text/html"
	} else if strings.HasSuffix(fileName, ".css") {
		mime = "text/css"
	} else if strings.HasSuffix(fileName, ".txt") {
		mime = "text/plain"
	} else if strings.HasSuffix(fileName, ".ttf") {
		mime = "font/ttf"
	} else if strings.HasSuffix(fileName, ".woff") {
		mime = "font/woff"
	} else if strings.HasSuffix(fileName, ".woff2") {
		mime = "font/woff2"
	} else if strings.HasSuffix(fileName, ".ico") {
		mime = "image/x-icon"
	} else if strings.HasSuffix(fileName, ".jpeg") {
		mime = "image/jpeg"
	} else if strings.HasSuffix(fileName, ".jpg") {
		mime = "image/jpeg"
	} else if strings.HasSuffix(fileName, ".png") {
		mime = "image/png"
	} else if strings.HasSuffix(fileName, ".gif") {
		mime = "image/gif"
	} else if strings.HasSuffix(fileName, ".bmp") {
		mime = "image/bmp"
	} else if strings.HasSuffix(fileName, ".svg") {
		mime = "image/svg+xml"
	} else if strings.HasSuffix(fileName, ".tif") {
		mime = "image/tiff"
	} else if strings.HasSuffix(fileName, ".tiff") {
		mime = "image/tiff"
	} else if strings.HasSuffix(fileName, ".js") {
		mime = "text/javascript"
	} else if strings.HasSuffix(fileName, ".json") {
		mime = "application/json"
	} else if strings.HasSuffix(fileName, ".pdf") {
		mime = "application/pdf"
	} else if strings.HasSuffix(fileName, ".avi") {
		mime = "video/x-msvideo"
	} else if strings.HasSuffix(fileName, ".mp4") {
		mime = "video/mp4"
	} else if strings.HasSuffix(fileName, ".mpeg") {
		mime = "video/mpeg"
	} else if strings.HasSuffix(fileName, ".ogv") {
		mime = "video/ogg"
	} else if strings.HasSuffix(fileName, ".webm") {
		mime = "video/webm"
	} else if strings.HasSuffix(fileName, ".jpgv") {
		mime = "video/jpg"
	} else if strings.HasSuffix(fileName, ".wasm") {
		mime = "application/wasm"
	} else if strings.HasSuffix(fileName, ".mkv") {
		mime = "video/x-matroska"
	} else if strings.HasSuffix(fileName, ".csv") {
		mime = "text/csv"
	} else if strings.HasSuffix(fileName, ".ics") {
		mime = "text/calendar"
	} else if strings.HasSuffix(fileName, ".sh") {
		mime = "application/x-sh"
	} else if strings.HasSuffix(fileName, ".swf") {
		mime = "application/x-shockwave-flash"
	} else if strings.HasSuffix(fileName, ".tar") {
		mime = "application/x-tar"
	} else if strings.HasSuffix(fileName, ".xls") {
		mime = "application/vnd.ms-excel"
	} else if strings.HasSuffix(fileName, ".xml") {
		mime = "application/xml"
	} else if strings.HasSuffix(fileName, ".xul") {
		mime = "application/vnd.mozilla.xul+xml"
	} else if strings.HasSuffix(fileName, ".zip") {
		mime = "application/zip"
	} else if strings.HasSuffix(fileName, ".7z") {
		mime = "application/x-7z-compressed"
	} else if strings.HasSuffix(fileName, ".apk") {
		mime = "application/vnd.android.package-archive"
	} else if strings.HasSuffix(fileName, ".jar") {
		mime = "application/java-archive"
	} else if strings.HasSuffix(fileName, ".vsd") {
		mime = "application/vnd.visio"
	} else if strings.HasSuffix(fileName, ".xhtml") {
		mime = "application/xhtml+xml"
	} else if strings.HasSuffix(fileName, ".mpkg") {
		mime = "application/vnd.apple.installer+xml"
	} else if strings.HasSuffix(fileName, ".ppt") {
		mime = "application/vnd.ms-powerpoint"
	} else if strings.HasSuffix(fileName, ".rar") {
		mime = "application/x-rar-compressed"
	} else if strings.HasSuffix(fileName, ".rtf") {
		mime = "application/rtf"
	} else if strings.HasSuffix(fileName, ".3gp") {
		mime = "video/3gpp"
	} else if strings.HasSuffix(fileName, ".wav") {
		mime = "audio/x-wav"
	} else if strings.HasSuffix(fileName, ".weba") {
		mime = "audio/webm"
	} else if strings.HasSuffix(fileName, ".mp3") {
		mime = "audio/mpeg"
	} else if strings.HasSuffix(fileName, ".3g2") {
		mime = "video/3gpp2"
	} else if strings.HasSuffix(fileName, ".aac") {
		mime = "audio/aac"
	} else if strings.HasSuffix(fileName, ".midi") {
		mime = "audio/midi"
	} else if strings.HasSuffix(fileName, ".mid") {
		mime = "audio/midi"
	} else if strings.HasSuffix(fileName, ".oga") {
		mime = "audio/og"
	} else if strings.HasSuffix(fileName, ".abw") {
		mime = "application/x-abiword"
	} else if strings.HasSuffix(fileName, ".arc") {
		mime = "application/octet-stream"
	} else if strings.HasSuffix(fileName, ".azw") {
		mime = "application/vnd.amazon.ebook"
	} else if strings.HasSuffix(fileName, ".bin") {
		mime = "application/octet-stream"
	} else if strings.HasSuffix(fileName, ".bz") {
		mime = "application/x-bzip"
	} else if strings.HasSuffix(fileName, ".bz2") {
		mime = "application/x-bzip2"
	} else if strings.HasSuffix(fileName, ".csh") {
		mime = "application/x-csh"
	} else if strings.HasSuffix(fileName, ".doc") {
		mime = "application/msword"
	} else if strings.HasSuffix(fileName, ".epub") {
		mime = "application/epub+zip"
	} else if strings.HasSuffix(fileName, ".odp") {
		mime = "application/vnd.oasis.opendocument.presentation"
	} else if strings.HasSuffix(fileName, ".ods") {
		mime = "application/vnd.oasis.opendocument.spreadsheet"
	} else if strings.HasSuffix(fileName, ".odt") {
		mime = "application/vnd.oasis.opendocument.text"
	} else if strings.HasSuffix(fileName, ".ogx") {
		mime = "application/ogg"
	}

	return mime
}
