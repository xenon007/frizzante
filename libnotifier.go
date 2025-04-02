package frizzante

import (
	"fmt"
	"os"
)

type Notifier struct {
	errorFile   *os.File
	messageFile *os.File
}

// NotifierCreate creates a notifier.
func NotifierCreate() *Notifier {
	return &Notifier{
		errorFile:   os.Stderr,
		messageFile: os.Stdout,
	}
}

// NotifierSendError sends an error to the notifier.
func NotifierSendError(self *Notifier, err error) {
	_, errorLocal := self.errorFile.WriteString(err.Error() + "\n")
	if errorLocal != nil {
		fmt.Printf("notifier could not write to error file")
	}
}

// NotifierSendMessage sends a message to the notifier.
func NotifierSendMessage(self *Notifier, message string) {
	_, errorLocal := self.messageFile.WriteString(message + "\n")
	if errorLocal != nil {
		fmt.Printf("notifier could not write to message file")
	}
}
