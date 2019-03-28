package manager

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"

	"github.com/skycoin/skywire/internal/httputil"
)

const (
	sessionCookieName = "swm-session"
)

// Errors associated with user management.
var (
	ErrBadBody           = errors.New("ill-formatted request body")
	ErrNotLoggedOut      = errors.New("not logged out")
	ErrBadLogin          = errors.New("incorrect username or password")
	ErrBadSession        = errors.New("session cookie is either non-existent, expired, or ill-formatted")
	ErrBadUsernameFormat = errors.New("format of 'username' is not accepted")
	ErrBadPasswordFormat = errors.New("format of 'password' is not accepted")
	ErrUserNotCreated    = errors.New("failed to create new user: username is either already taken, or unaccepted")
	ErrUserNotFound      = errors.New("user is either deleted or not found")
)

// for use with context.Context
type ctxKey string

const (
	userKey    = ctxKey("user")
	sessionKey = ctxKey("session")
)

// Session represents a user session.
type Session struct {
	SID    uuid.UUID `json:"sid"`
	User   string    `json:"username"`
	Expiry time.Time `json:"expiry"`
}

// UserManager manages the users and sessions.
type UserManager struct {
	c        CookieConfig
	db       UserStore
	sessions map[uuid.UUID]Session
	crypto   *securecookie.SecureCookie
	mu       *sync.RWMutex
}

// NewUserManager creates a new UserManager.
func NewUserManager(users UserStore, config CookieConfig) *UserManager {
	return &UserManager{
		db:       users,
		c:        config,
		sessions: make(map[uuid.UUID]Session),
		crypto:   securecookie.New(config.HashKey, config.BlockKey),
		mu:       new(sync.RWMutex),
	}
}

// Login returns a HandlerFunc for login operations.
func (s *UserManager) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := s.session(r); ok {
			httputil.WriteJSON(w, r, http.StatusForbidden, ErrNotLoggedOut)
			return
		}
		var rb struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := httputil.ReadJSON(r, &rb); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, ErrBadBody)
			return
		}
		user, ok := s.db.User(rb.Username)
		if !ok || !user.VerifyPassword(rb.Password) {
			httputil.WriteJSON(w, r, http.StatusUnauthorized, ErrBadLogin)
			return
		}
		s.newSession(w, Session{
			User:   rb.Username,
			Expiry: time.Now().Add(s.c.ExpiresDuration),
		})
		//http.SetCookie()
		httputil.WriteJSON(w, r, http.StatusOK, ok)
	}
}

// Logout returns a HandlerFunc of logout operations.
func (s *UserManager) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.delSession(w, r); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, errors.New("not logged in"))
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	}
}

// Authorize is an http middleware for authorizing requests.
func (s *UserManager) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session, ok := s.session(r)
		if !ok {
			httputil.WriteJSON(w, r, http.StatusUnauthorized, ErrBadSession)
			return
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, userKey, user)
		ctx = context.WithValue(ctx, sessionKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ChangePassword returns a HandlerFunc for changing the user's password.
func (s *UserManager) ChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			user = r.Context().Value(userKey).(User)
		)
		var rb struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}
		if err := httputil.ReadJSON(r, &rb); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		if ok := user.VerifyPassword(rb.OldPassword); !ok {
			httputil.WriteJSON(w, r, http.StatusUnauthorized, ErrBadLogin)
			return
		}
		if ok := user.SetPassword(rb.NewPassword); !ok {
			httputil.WriteJSON(w, r, http.StatusBadRequest, ErrBadPasswordFormat)
			return
		}
		if ok := s.db.SetUser(user); !ok {
			httputil.WriteJSON(w, r, http.StatusForbidden, ErrUserNotFound)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	}
}

// CreateAccount returns a HandlerFunc for account creation.
func (s *UserManager) CreateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rb struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := httputil.ReadJSON(r, &rb); err != nil {
			httputil.WriteJSON(w, r, http.StatusBadRequest, err)
			return
		}
		var user User
		if ok := user.SetName(rb.Username); !ok {
			httputil.WriteJSON(w, r, http.StatusBadRequest, ErrBadUsernameFormat)
			return
		}
		if ok := user.SetPassword(rb.Password); !ok {
			httputil.WriteJSON(w, r, http.StatusBadRequest, ErrBadPasswordFormat)
			return
		}
		if ok := s.db.AddUser(user); !ok {
			httputil.WriteJSON(w, r, http.StatusForbidden, ErrUserNotCreated)
			return
		}
		httputil.WriteJSON(w, r, http.StatusOK, true)
	}
}

// UserInfo returns a HandlerFunc for obtaining user info.
func (s *UserManager) UserInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			user    = r.Context().Value(userKey).(User)
			session = r.Context().Value(sessionKey).(Session)
		)
		var otherSessions []Session
		s.mu.RLock()
		for _, s := range s.sessions {
			if s.User == user.Name && s.SID != session.SID {
				otherSessions = append(otherSessions, s)
			}
		}
		s.mu.RUnlock()
		httputil.WriteJSON(w, r, http.StatusOK, struct {
			Username string    `json:"username"`
			Current  Session   `json:"current_session"`
			Sessions []Session `json:"other_sessions"`
		}{
			Username: user.Name,
			Current:  session,
			Sessions: otherSessions,
		})
	}
}

func (s *UserManager) newSession(w http.ResponseWriter, session Session) {
	session.SID = uuid.New()
	s.mu.Lock()
	s.sessions[session.SID] = session
	s.mu.Unlock()
	value, err := s.crypto.Encode(sessionCookieName, session.SID)
	catch(err)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Domain:   s.c.Domain,
		Expires:  time.Now().Add(s.c.ExpiresDuration),
		Secure:   s.c.Secure,
		HttpOnly: s.c.HTTPOnly,
		SameSite: s.c.SameSite,
	})
}

func (s *UserManager) delSession(w http.ResponseWriter, r *http.Request) error {
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
		Name:     sessionCookieName,
		Domain:   s.c.Domain,
		MaxAge:   -1,
		Secure:   s.c.Secure,
		HttpOnly: s.c.HTTPOnly,
		SameSite: s.c.SameSite,
	})
	return nil
}

func (s *UserManager) session(r *http.Request) (User, Session, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return User{}, Session{}, false
	}
	var sid uuid.UUID
	if err := s.crypto.Decode(sessionCookieName, cookie.Value, &sid); err != nil {
		log.WithError(err).Warn("failed to decode session cookie value")
		return User{}, Session{}, false
	}
	s.mu.RLock()
	session, ok := s.sessions[sid]
	s.mu.RUnlock()
	if !ok {
		return User{}, Session{}, false
	}
	user, ok := s.db.User(session.User)
	if !ok {
		return User{}, Session{}, false
	}
	if time.Now().After(session.Expiry) {
		s.mu.Lock()
		delete(s.sessions, sid)
		s.mu.Unlock()
		return User{}, Session{}, false
	}
	return user, session, true
}
