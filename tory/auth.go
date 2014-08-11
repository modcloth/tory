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
	r.Header.Set("Tory-Authorized", "nope")

	authHeader := r.Header.Get("Authorization")
	if strings.TrimSpace(authHeader) == fmt.Sprintf("token %s", a.Token) {
		r.Header.Set("Tory-Authorized", "yep")
	}

	next(w, r)
}
