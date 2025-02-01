package frizzante

import "github.com/nu7hatch/gouuid"

var sessions = map[string]*Session{}

type Session struct {
	id   string
	data map[string]interface{}
}

func (s *Session) get(key string, defaultValue interface{}) interface{} {
	sessionItem, ok := s.data[key]
	if !ok {
		s.data[key] = defaultValue
		return s.data[key]
	}

	return sessionItem
}

func (s *Session) set(key string, defaultValue interface{}) {
	s.data[key] = defaultValue
}

func sessionCreate() (*Session, error) {
	sessionId, uuidError := uuid.NewV4()
	if uuidError != nil {
		return nil, uuidError
	}

	return &Session{
		id:   sessionId.String(),
		data: map[string]interface{}{},
	}, nil
}

// SessionStart starts a new session or retrieves it if it's already been started.
//
// SessionStart always returns two functions, a get() and a set(), which you can use to manage the session.
func SessionStart(request *Request, response *Response) (
	get func(key string, defaultValue interface{}) interface{},
	set func(key string, defaultValue interface{}),
) {
	sessionIdCookie, cookieError := request.HttpRequest.Cookie("session-id")
	if cookieError != nil {
		freshSession, sessionError := sessionCreate()
		if sessionError != nil {
			ServerNotifyError(request.server, sessionError)
			return nil, nil
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = freshSession.get
		set = freshSession.set
		return
	}

	session, sessionExists := sessions[sessionIdCookie.Value]
	if !sessionExists {
		freshSession, sessionError := sessionCreate()
		if sessionError != nil {
			ServerNotifyError(request.server, sessionError)
			return nil, nil
		}

		SendCookie(response, "session-id", freshSession.id)
		sessions[freshSession.id] = freshSession
		get = freshSession.get
		set = freshSession.set
		return
	}

	SendCookie(response, "session-id", sessionIdCookie.Value)
	get = session.get
	set = session.set
	return
}

// SessionDestroy destroys the session.
func SessionDestroy(self *Session) {
	delete(sessions, self.id)
}
