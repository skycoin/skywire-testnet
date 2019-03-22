package manager

import (
	"encoding/gob"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/skycoin/skywire/internal/httputil"
	"net/http"
	"sync"
	"time"
)

const (
	sessionCookieName = "swm_session"
)

func init() {
	gob.Register(uuid.UUID{})
}

type Session struct {
	User   string
	Expiry time.Time
}

type SessionsConfig struct {
	HashKey  []byte
	BlockKey []byte

	Path     string    // optional
	Domain   string    // optional
	Expires  time.Time // optional
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type SessionsManager struct {
	users UserStorer

	config   SessionsConfig
	sessions map[uuid.UUID]*Session
	crypto   *securecookie.SecureCookie
	mu       *sync.RWMutex
}

func NewSessionsManager(users UserStorer, config SessionsConfig) *SessionsManager {
	return &SessionsManager{
		users:    users,
		config:   config,
		sessions: make(map[uuid.UUID]*Session),
		crypto:   securecookie.New(config.HashKey, config.BlockKey),
		mu:       new(sync.RWMutex),
	}
}

func (s *SessionsManager) Login() http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie(sessionCookieName); err != http.ErrNoCookie {
			httputil.WriteJSON(w, r, http.StatusBadRequest, errors.New("not logged out"))
			return
		}
		var req Request
		if err := httputil.ReadJSON(r, &req); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, errors.New("cannot read request body"))
			return
		}
		ok := s.users.VerifyPassword(req.Username, req.Password)
		if !ok {
			httputil.WriteJSON(w, r, http.StatusUnauthorized, errors.New("incorrect username or password"))
			return
		}
		s.newSession(w, &Session{
			User:   req.Username,
			Expiry: time.Now().Add(time.Hour * 12), // TODO: Set default expiry.
		})
		httputil.WriteJSON(w, r, http.StatusOK, ok)
	}
}

func (s *SessionsManager) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.delSession(w, r); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, errors.New("not logged in"))
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	}
}

func (s *SessionsManager) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.checkSession(r); err != nil {
			httputil.WriteJSON(w, r, http.StatusUnauthorized, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *SessionsManager) newSession(w http.ResponseWriter, session *Session) {
	sid := uuid.New()
	s.mu.Lock()
	s.sessions[sid] = session
	s.mu.Unlock()
	value, err := s.crypto.Encode(sessionCookieName, sid)
	catch(err)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Domain:   s.config.Domain,
		Expires:  s.config.Expires,
		Secure:   s.config.Secure,
		HttpOnly: s.config.HttpOnly,
		SameSite: s.config.SameSite,
	})
}

func (s *SessionsManager) delSession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return err
	}
	var sid uuid.UUID
	if err := s.crypto.Decode(sessionCookieName, cookie.Value, &sid); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.sessions, sid)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName,
		Domain:   s.config.Domain,
		MaxAge:   -1,
		Secure:   s.config.Secure,
		HttpOnly: s.config.HttpOnly,
		SameSite: s.config.SameSite,
	})
	return nil
}

func (s *SessionsManager) checkSession(r *http.Request) error {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return err
	}
	var sid uuid.UUID
	if err := s.crypto.Decode(sessionCookieName, cookie.Value, &sid); err != nil {
		return err
	}
	s.mu.RLock()
	session, ok := s.sessions[sid]
	s.mu.RUnlock()
	if !ok {
		return errors.New("invalid session") // TODO: proper error
	}
	if time.Now().After(session.Expiry) {
		s.mu.Lock()
		delete(s.sessions, sid)
		s.mu.Unlock()
		return errors.New("invalid session") // TODO: proper error
	}
	return nil
}