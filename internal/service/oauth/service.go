package oauth

import (
	"log/slog"
	"net/http"
	"strings"

	"arktie.org/internal/data"
	"arktie.org/internal/lib/libhttp"
	"arktie.org/internal/lib/liblogs"

	"github.com/bluesky-social/indigo/atproto/identity"
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
		http.Redirect(w, r, "/?error="+err.Error(), http.StatusFound)
		return
	}

	atIdent, err := identity.DefaultDirectory().LookupDID(r.Context(), atSession.AccountDID)
	if err != nil {
		slog.WarnContext(r.Context(), "failed to lookup DID", liblogs.ErrAttr(err), slog.Any("at_session", atSession))
	}

	_, _ = atSession, atIdent

	http.Redirect(w, r, "/", http.StatusFound)
}
