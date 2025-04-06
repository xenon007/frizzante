package frizzante

import (
	"embed"
	"errors"
	"os"
	"path/filepath"
)

// EmbeddedExists checks if file exists.
func EmbeddedExists(embeddedFileSystem embed.FS, fileName string) bool {
	return EmbeddedIsFile(embeddedFileSystem, fileName) || EmbeddedIsDirectory(embeddedFileSystem, fileName)
}

// EmbeddedIsFile check if file exists and is a file.
func EmbeddedIsFile(embeddedFileSystem embed.FS, fileName string) bool {
	_, err := embeddedFileSystem.ReadFile(fileName)
	if err != nil {
		return false
	}
	return true
}

// EmbeddedIsDirectory checks if file exists and is a directory.
func EmbeddedIsDirectory(embeddedFileSystem embed.FS, fileName string) bool {
	_, err := embeddedFileSystem.ReadDir(fileName)
	if err != nil {
		return false
	}
	return true
}

// Exists checks if file exists.
func Exists(fileName string) bool {
	_, statError := os.Stat(fileName)
	return nil == statError || !errors.Is(statError, os.ErrNotExist)
}

// Delete deletes a file.
func Delete(fileName string) bool {
	removeError := os.Remove(fileName)
	return nil == removeError || !errors.Is(removeError, os.ErrNotExist)
}

// IsFile check if file exists and is a file.
func IsFile(fileName string) bool {
	stat, statError := os.Stat(fileName)
	if statError != nil {
		return !errors.Is(statError, os.ErrNotExist)
	}
	return !stat.IsDir()
}

// IsDirectory checks if file exists and is a directory.
func IsDirectory(fileName string) bool {
	stat, statError := os.Stat(fileName)
	if statError != nil {
		return !errors.Is(statError, os.ErrNotExist)
	}
	return stat.IsDir()
}

var mimes = map[string]string{
	".html":  "text/html",
	".css":   "text/css",
	".txt":   "text/plain",
	".ttf":   "font/ttf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ico":   "image/x-icon",
	".jpeg":  "image/jpeg",
	".jpg":   "image/jpeg",
	".png":   "image/png",
	".gif":   "image/gif",
	".bmp":   "image/bmp",
	".svg":   "image/svg+xml",
	".tif":   "image/tiff",
	".tiff":  "image/tiff",
	".js":    "text/javascript",
	".json":  "application/json",
	".pdf":   "application/pdf",
	".avi":   "video/x-msvideo",
	".mp4":   "video/mp4",
	".mpeg":  "video/mpeg",
	".ogv":   "video/ogg",
	".webm":  "video/webm",
	".jpgv":  "video/jpg",
	".wasm":  "application/wasm",
	".mkv":   "video/x-matroska",
	".csv":   "text/csv",
	".ics":   "text/calendar",
	".sh":    "application/x-sh",
	".swf":   "application/x-shockwave-flash",
	".tar":   "application/x-tar",
	".xls":   "application/vnd.ms-excel",
	".xml":   "application/xml",
	".xul":   "application/vnd.mozilla.xul+xml",
	".zip":   "application/zip",
	".7z":    "application/x-7z-compressed",
	".apk":   "application/vnd.android.package-archive",
	".jar":   "application/java-archive",
	".vsd":   "application/vnd.visio",
	".xhtml": "application/xhtml+xml",
	".mpkg":  "application/vnd.apple.installer+xml",
	".ppt":   "application/vnd.ms-powerpoint",
	".rar":   "application/x-rar-compressed",
	".rtf":   "application/rtf",
	".3gp":   "video/3gpp",
	".wav":   "audio/x-wav",
	".weba":  "audio/webm",
	".mp3":   "audio/mpeg",
	".3g2":   "video/3gpp2",
	".aac":   "audio/aac",
	".midi":  "audio/midi",
	".mid":   "audio/midi",
	".oga":   "audio/og",
	".abw":   "application/x-abiword",
	".arc":   "application/octet-stream",
	".azw":   "application/vnd.amazon.ebook",
	".bin":   "application/octet-stream",
	".bz":    "application/x-bzip",
	".bz2":   "application/x-bzip2",
	".csh":   "application/x-csh",
	".doc":   "application/msword",
	".epub":  "application/epub+zip",
	".odp":   "application/vnd.oasis.opendocument.presentation",
	".ods":   "application/vnd.oasis.opendocument.spreadsheet",
	".odt":   "application/vnd.oasis.opendocument.text",
	".ogx":   "application/ogg",
}

func Mime(fileName string) string {
	extensionName := filepath.Ext(fileName)
	mime, ok := mimes[extensionName]

	if ok {
		return mime
	}

	return "text/plain"
}
