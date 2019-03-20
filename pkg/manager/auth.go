package manager

import (
	"net/http"
	"time"
)

const (
	sessionCookieName = "swm_session"

	boltTimeout           = 10 * time.Second
	boltUserBucketName    = "users"
	boltSessionBucketName = "sessions"
)

type CookieConfig struct {
	Domain   string // optional
	MaxAge   int
	HTTPOnly bool
	Secure   bool
	SameSite http.SameSite
}

type AuthConfig struct {
	StorePath string
}

type Auth struct {
}

func NewAuth(c AuthConfig) (*Auth, error) {
	return nil, nil
}

func (a *Auth) ChangePassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (a *Auth) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	}
}

func (a *Auth) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
