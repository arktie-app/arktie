package oauth

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/libhttp"
	"arktie.org/internal/lib/libjwt"
	"arktie.org/internal/lib/liblogs"
	"arktie.org/internal/server/middleware"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	cfg    *data.Config
	client *data.Client
}

func NewService(cfg *data.Config, client *data.Client) *Service {
	return &Service{
		cfg:    cfg,
		client: client,
	}
}

// ClientConfig handles GET /oauth/client-metadata.json
func (svc *Service) ClientConfig(w http.ResponseWriter, r *http.Request) {
	libhttp.WriteJSON(w, http.StatusOK, svc.client.OAuth.Config.ClientMetadata())
}

// Start handles GET /oauth/start?handle=<handle>.
//
// Resolves the user's PDS, initiates a PAR request, and returns the
// authorization redirect URL for the SPA to navigate the user to.
func (svc *Service) Start(w http.ResponseWriter, r *http.Request) {
	handle := strings.TrimSpace(r.URL.Query().Get("handle"))
	if handle == "" {
		libhttp.WriteError(w, http.StatusBadRequest, "missing required query parameter: handle")
		return
	}

	redirectURL, err := svc.client.OAuth.StartAuthFlow(r.Context(), handle)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to start auth flow", liblogs.ErrAttr(err))
		libhttp.WriteError(w, http.StatusBadRequest, "failed to start auth flow")
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// Callback handles GET /oauth/callback.
//
// The PDS redirects the user's browser here with ?code=...&state=... after
// authorization. API verifies the state (via indigo), performs the token
// exchange, creates a server-side session, sets the httpOnly cookie, and
// redirects the browser to the home page.
func (svc *Service) Callback(w http.ResponseWriter, r *http.Request) {
	atSession, err := svc.client.OAuth.ProcessCallback(r.Context(), r.URL.Query())
	if err != nil {
		slog.WarnContext(r.Context(), "failed to get OAuth callback", liblogs.ErrAttr(err))
		http.Redirect(w, r, "/?error=process_callback_failed", http.StatusFound)
		return
	}

	atIdent, err := identity.DefaultDirectory().LookupDID(r.Context(), atSession.AccountDID)
	if err != nil {
		slog.WarnContext(r.Context(), "failed to lookup DID", liblogs.ErrAttr(err), slog.Any("at_session", atSession))
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		libjwt.NewClaim(svc.cfg).
			WithATSession(atSession).
			WithATIdentity(atIdent),
	)

	tokenString, err := token.SignedString(svc.cfg.App.Key)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to sign JWT", liblogs.ErrAttr(err))
		http.Redirect(w, r, "/?error=sign_jwt_failed", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     data.SessionCookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   !svc.cfg.App.IsLocal(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().AddDate(0, 0, 14),
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (svc *Service) Logout(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	if user != nil {
		if err := svc.client.OAuth.Logout(r.Context(), user.Session.AccountDID, user.Session.SessionID); err != nil {
			slog.WarnContext(r.Context(), "failed to logout from oauth client", liblogs.ErrAttr(err))
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     data.SessionCookieName,
		Path:     "/",
		HttpOnly: true,
		Secure:   !svc.cfg.App.IsLocal(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusNoContent)
}
