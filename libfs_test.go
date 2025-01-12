package frizzante

import (
	"embed"
	"testing"
)

//go:embed libfs.go
//go:embed .github
//go:embed www/dist/*/**
var embeddedFileSystem embed.FS

func TestEmbeddedExists(test *testing.T) {
	// Positive.
	fileName := "libfs.go"
	actual := EmbeddedExists(embeddedFileSystem, fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to exist", fileName)
	}

	// Negative.
	fileName = "qwerty"
	actual = EmbeddedExists(embeddedFileSystem, fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to exist", fileName)
	}
}

func TestEmbeddedIsFile(test *testing.T) {
	// Positive.
	fileName := "libfs.go"
	actual := EmbeddedIsFile(embeddedFileSystem, fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to be a file", fileName)
	}

	// Negatives.
	fileName = ".github"
	actual = EmbeddedIsFile(embeddedFileSystem, fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to not be a file", fileName)
	}

	fileName = "qwerty"
	actual = EmbeddedIsFile(embeddedFileSystem, fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to not be a file", fileName)
	}
}

func TestEmbeddedIsDirectory(test *testing.T) {
	// Positive.
	fileName := ".github"
	actual := EmbeddedIsDirectory(embeddedFileSystem, fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to be a directory", fileName)
	}

	// Negatives.
	fileName = "libfs.go"
	actual = EmbeddedIsDirectory(embeddedFileSystem, fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to not be a directory", fileName)
	}

	fileName = "qwerty"
	actual = EmbeddedIsDirectory(embeddedFileSystem, fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s (embedded) was expected to not be a directory", fileName)
	}
}

func TestExists(test *testing.T) {
	// Positive.
	fileName := "libfs.go"
	actual := Exists(fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s was expected to exist", fileName)
	}

	// Negative.
	fileName = "qwerty"
	actual = Exists(fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s was expected to not exist", fileName)
	}
}

func TestIsFile(test *testing.T) {
	// Positive.
	fileName := "libfs.go"
	actual := IsFile(fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s was expected to be a file", fileName)
	}

	// Negatives.
	fileName = ".github"
	actual = IsFile(fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s was expected to not be a file", fileName)
	}

	fileName = "qwerty"
	actual = IsFile(fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s was expected to not be a file", fileName)
	}
}

func TestIsDirectory(test *testing.T) {
	// Positive.
	fileName := ".github"
	actual := IsDirectory(fileName)
	expected := true
	if actual != expected {
		test.Fatalf("%s was expected to be a directory", fileName)
	}

	// Negatives.
	fileName = "libfs.go"
	actual = IsDirectory(fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s was expected to not be a directory", fileName)
	}

	fileName = "qwerty"
	actual = IsDirectory(fileName)
	expected = false
	if actual != expected {
		test.Fatalf("%s was expected to not be a directory", fileName)
	}
}

var expectedMimes = map[string]string{
	"my.file.html":  "text/html",
	"my.file.css":   "text/css",
	"my.file.txt":   "text/plain",
	"my.file.ttf":   "font/ttf",
	"my.file.woff":  "font/woff",
	"my.file.woff2": "font/woff2",
	"my.file.ico":   "image/x-icon",
	"my.file.jpeg":  "image/jpeg",
	"my.file.jpg":   "image/jpeg",
	"my.file.png":   "image/png",
	"my.file.gif":   "image/gif",
	"my.file.bmp":   "image/bmp",
	"my.file.svg":   "image/svg+xml",
	"my.file.tif":   "image/tiff",
	"my.file.tiff":  "image/tiff",
	"my.file.js":    "text/javascript",
	"my.file.json":  "application/json",
	"my.file.pdf":   "application/pdf",
	"my.file.avi":   "video/x-msvideo",
	"my.file.mp4":   "video/mp4",
	"my.file.mpeg":  "video/mpeg",
	"my.file.ogv":   "video/ogg",
	"my.file.webm":  "video/webm",
	"my.file.jpgv":  "video/jpg",
	"my.file.wasm":  "application/wasm",
	"my.file.mkv":   "video/x-matroska",
	"my.file.csv":   "text/csv",
	"my.file.ics":   "text/calendar",
	"my.file.sh":    "application/x-sh",
	"my.file.swf":   "application/x-shockwave-flash",
	"my.file.tar":   "application/x-tar",
	"my.file.xls":   "application/vnd.ms-excel",
	"my.file.xml":   "application/xml",
	"my.file.xul":   "application/vnd.mozilla.xul+xml",
	"my.file.zip":   "application/zip",
	"my.file.7z":    "application/x-7z-compressed",
	"my.file.apk":   "application/vnd.android.package-archive",
	"my.file.jar":   "application/java-archive",
	"my.file.vsd":   "application/vnd.visio",
	"my.file.xhtml": "application/xhtml+xml",
	"my.file.mpkg":  "application/vnd.apple.installer+xml",
	"my.file.ppt":   "application/vnd.ms-powerpoint",
	"my.file.rar":   "application/x-rar-compressed",
	"my.file.rtf":   "application/rtf",
	"my.file.3gp":   "video/3gpp",
	"my.file.wav":   "audio/x-wav",
	"my.file.weba":  "audio/webm",
	"my.file.mp3":   "audio/mpeg",
	"my.file.3g2":   "video/3gpp2",
	"my.file.aac":   "audio/aac",
	"my.file.midi":  "audio/midi",
	"my.file.mid":   "audio/midi",
	"my.file.oga":   "audio/og",
	"my.file.abw":   "application/x-abiword",
	"my.file.arc":   "application/octet-stream",
	"my.file.azw":   "application/vnd.amazon.ebook",
	"my.file.bin":   "application/octet-stream",
	"my.file.bz":    "application/x-bzip",
	"my.file.bz2":   "application/x-bzip2",
	"my.file.csh":   "application/x-csh",
	"my.file.doc":   "application/msword",
	"my.file.epub":  "application/epub+zip",
	"my.file.odp":   "application/vnd.oasis.opendocument.presentation",
	"my.file.ods":   "application/vnd.oasis.opendocument.spreadsheet",
	"my.file.odt":   "application/vnd.oasis.opendocument.text",
	"my.file.ogx":   "application/ogg",
}

func TestMime(test *testing.T) {
	// Positives.
	for fileName, expected := range expectedMimes {
		actual := Mime(fileName)
		if actual != expected {
			test.Fatalf("file %s was expected to resolve into mime %s, received %s instead", fileName, expected, actual)
		}
	}

	// Negative.
	fileName := "my.file.qwerty123"
	actual := Mime(fileName)
	expected := "text/plain"
	if actual != expected {
		test.Fatalf("file %s was expected to resolve into mime %s, received %s instead", fileName, expected, actual)
	}
}
