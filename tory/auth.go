package tory

import (
	"fmt"
	"net/http"
	"strings"
)

type authMiddleware struct {
	Token string
}

func newAuthMiddleware(token string) *authMiddleware {
	return &authMiddleware{Token: token}
}

func (a *authMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	r.Header.Set("Tory-Authenticated", "nope")

	authHeader := r.Header.Get("Authentication")
	if strings.TrimSpace(authHeader) == fmt.Sprintf("token %s", a.Token) {
		r.Header.Set("Tory-Authenticated", "yep")
	}

	next(w, r)
}
