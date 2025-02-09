package frizzante

import uuid "github.com/nu7hatch/gouuid"

var sessions = map[string]*Session{}

type Session struct {
	id      string
	get     func(key string, defaultValue any) (value any)
	set     func(key string, value any)
	unset   func(key string)
	destroy func()
}

// SessionStart starts a new session or retrieves it if it's already been started.
//
// SessionStart always returns two functions, a get() and a set(), which you can use to manage the session.
func SessionStart(request *Request, response *Response) (
	get func(key string, defaultValue any) (value any),
	set func(key string, value any),
	unset func(key string),
) {
	sessionIdCookie, cookieError := request.HttpRequest.Cookie("session-id")
	if cookieError != nil {
		uuidV4, sessionIdError := uuid.NewV4()

		if sessionIdError != nil {
			ServerNotifyError(request.server, sessionIdError)
		}

		sessionId := uuidV4.String()

		sessionGet, sessionSet, sessionUnset, sessionDestroy := request.server.sessionHandler(sessionId)

		freshSession := &Session{
			id:      sessionId,
			get:     sessionGet,
			set:     sessionSet,
			unset:   sessionUnset,
			destroy: sessionDestroy,
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = sessionGet
		set = sessionSet
		return
	}

	session, sessionExists := sessions[sessionIdCookie.Value]
	if !sessionExists {
		uuidV4, sessionIdError := uuid.NewV4()
		if sessionIdError != nil {
			ServerNotifyError(request.server, sessionIdError)
		}
		sessionId := uuidV4.String()
		sessionGet, sessionSet, sessionUnset, sessionDestroy := request.server.sessionHandler(sessionId)

		freshSession := &Session{
			id:      sessionId,
			get:     sessionGet,
			set:     sessionSet,
			unset:   sessionUnset,
			destroy: sessionDestroy,
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = sessionGet
		set = sessionSet
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
