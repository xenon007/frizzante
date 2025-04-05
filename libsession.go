package frizzante

import (
	uuid "github.com/nu7hatch/gouuid"
	"net/http"
)

var sessions = map[string]*Session{}

type Session struct {
	id       string
	get      func(key string, defaultValue any) (value any)
	set      func(key string, value any)
	unset    func(key string)
	validate func() (valid bool)
	destroy  func()
}

// SessionStart first tries to retrieve the client session, then,
// if the client is not associated with any existing session,
// it will automatically create a new empty session for said client.
//
// It always returns three functions, get, set and unset.
//
// Use get to retrieve a property from the session.
//
// Use set to create a new property or update an existing one to the session.
//
// Use unset to remove a property from the session.
func SessionStart(request *Request, response *Response) (
	get func(key string, defaultValue any) (value any),
	set func(key string, value any),
	unset func(key string),
) {

	var sessionIdCookie *http.Cookie
	sessionIdCookies := request.httpRequest.CookiesNamed("session-id")
	sessionIdCookiesLen := len(sessionIdCookies)

	if 0 == sessionIdCookiesLen {
		uuidV4, sessionIdError := uuid.NewV4()

		if sessionIdError != nil {
			NotifierSendError(request.server.notifier, sessionIdError)
		}

		sessionId := uuidV4.String()
		sessionGet,
			sessionSet,
			sessionUnset,
			sessionValidate,
			sessionDestroy := request.server.sessionOperator(sessionId)

		freshSession := &Session{
			id:       sessionId,
			get:      sessionGet,
			set:      sessionSet,
			unset:    sessionUnset,
			validate: sessionValidate,
			destroy:  sessionDestroy,
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = sessionGet
		set = sessionSet
		unset = sessionUnset
		return
	}

	var sessionExists bool
	var session *Session

	for _, cookie := range sessionIdCookies {
		session, sessionExists = sessions[cookie.Value]
		if sessionExists {
			sessionIdCookie = cookie
			break
		}
	}

	if !sessionExists {
		uuidV4, sessionIdError := uuid.NewV4()
		if sessionIdError != nil {
			NotifierSendError(request.server.notifier, sessionIdError)
		}
		sessionId := uuidV4.String()
		sessionGet,
			sessionSet,
			sessionUnset,
			sessionValidate,
			sessionDestroy := request.server.sessionOperator(sessionId)

		freshSession := &Session{
			id:       sessionId,
			get:      sessionGet,
			set:      sessionSet,
			unset:    sessionUnset,
			validate: sessionValidate,
			destroy:  sessionDestroy,
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = sessionGet
		set = sessionSet
		unset = sessionUnset
		return
	}

	if !session.validate() {
		delete(sessions, sessionIdCookie.Value)
		session.destroy()
		SessionStart(request, response)
		return
	}

	SendCookie(response, "session-id", session.id)
	get = session.get
	set = session.set
	unset = session.unset
	return
}

// SessionDestroy destroys the session.
func SessionDestroy(self *Session) {
	self.destroy()
}
