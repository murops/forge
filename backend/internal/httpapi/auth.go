package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"

	"github.com/murops/forge/backend/internal/requests"
)

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (requests.User, error)
}

type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCVerifier(ctx context.Context, issuer, audience string) (*OIDCVerifier, error) {
	if issuer == "" {
		return nil, errors.New("OIDC_ISSUER is required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return &OIDCVerifier{verifier: provider.Verifier(&oidc.Config{ClientID: audience})}, nil
}

func (v *OIDCVerifier) Verify(ctx context.Context, raw string) (requests.User, error) {
	tok, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return requests.User{}, err
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := tok.Claims(&claims); err != nil {
		return requests.User{}, err
	}
	return requests.User{Subject: tok.Subject, Email: claims.Email}, nil
}

const userKey = "user"

func requireAuth(v TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			abort(c, http.StatusUnauthorized, "missing bearer token")
			return
		}
		u, err := v.Verify(c.Request.Context(), raw)
		if err != nil {
			abort(c, http.StatusUnauthorized, "invalid token")
			return
		}
		c.Set(userKey, u)
		c.Next()
	}
}

func currentUser(c *gin.Context) requests.User {
	return c.MustGet(userKey).(requests.User)
}
