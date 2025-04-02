package frizzante

type Notifier struct {
	errorListeners   []func(error)
	messageListeners []func(string)
}

func NotifierCreate() *Notifier {
	return &Notifier{
		errorListeners:   []func(error){},
		messageListeners: []func(string){},
	}
}

func NotifierReceiveError(self *Notifier, callback func(err error)) {
	self.errorListeners = append(self.errorListeners, callback)
}

func NotifierReceiveMessage(self *Notifier, callback func(err string)) {
	self.messageListeners = append(self.messageListeners, callback)
}

func NotifierSendError(self *Notifier, err error) {
	for _, callback := range self.errorListeners {
		callback(err)
	}
}

func NotifierSendMessage(self *Notifier, message string) {
	for _, callback := range self.messageListeners {
		callback(message)
	}
}
