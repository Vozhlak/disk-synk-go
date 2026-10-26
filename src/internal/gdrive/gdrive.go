package gdrive

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Client struct {
	service *drive.Service
}

func New(ctx context.Context, clientID, clientSecret, tokenPath string) (*Client, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("google drive: client_id is required")
	}

	if strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("google drive: client_secret is required")
	}

	if strings.TrimSpace(tokenPath) == "" {
		return nil, errors.New("google drive: token_path is required")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  "http://127.0.0.1:8080/callback",
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveMetadataReadonlyScope},
	}

	token, err := loadOrAuthorizeToken(ctx, cfg, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("google drive client: authenticate: %w", err)
	}

	httpclient := cfg.Client(ctx, token)

	service, err := drive.NewService(ctx, option.WithHTTPClient(httpclient))
	if err != nil {
		return nil, fmt.Errorf("google drive client: create drive service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

func loadOrAuthorizeToken(ctx context.Context, cfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	token, err := loadToken(tokenPath)
	if err == nil {
		return token, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load saved token: %w", err)
	}

	return authorize(ctx, cfg, tokenPath)
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err = json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}

	return &token, nil
}

const callbackPath = "/callback"

func authorize(ctx context.Context, cfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate oauth state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: callbackHandler(state, codeCh, errCh),
	}

	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil &&
			!errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- fmt.Errorf("start local callback server: %w", serveErr)
		}
	}()

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)

	fmt.Printf(
		"Go to the following link in your browser:\n%s\n",
		authURL,
	)

	var code string

	select {
	case code = <-codeCh:
	case err = <-errCh:
		_ = server.Close()
		return nil, err
	case <-ctx.Done():
		_ = server.Close()
		return nil, fmt.Errorf("authorization canceled: %w", ctx.Err())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = server.Shutdown(shutdownCtx); err != nil {
		return nil, fmt.Errorf("shutdown local callback server: %w", err)
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}

	if err = saveToken(tokenPath, token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return token, nil
}

func callbackHandler(expectedState string, codeCh chan<- string, errCh chan<- error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}

		query := r.URL.Query()

		if errParam := query.Get("error"); errParam != "" {
			http.Error(w, "Google authorization failed", http.StatusBadRequest)
			errCh <- fmt.Errorf("google authorization error: %s", errParam)
			return
		}

		if query.Get("state") != expectedState {
			http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
			errCh <- errors.New("oauth state mismatch")
			return
		}

		code := query.Get("code")
		if code == "" {
			http.Error(w, "Authorization code is missing", http.StatusBadRequest)
			errCh <- errors.New("authorization code is missing")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<!doctype html>
			<html>
				<body>
					<h1>Authorization complete</h1>
					<p>You can close this tab and return to the terminal.</p>
				</body>
			</html>
		`))

		codeCh <- code
	})
}

func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", " ")
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}

	if err = os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	return nil
}

func generateState() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random state: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}

func (c *Client) CheckAccess(ctx context.Context) error {
	about, err := c.service.About.Get().Fields("user").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("check Google Drive access: %w", err)
	}

	if about.User == nil {
		return errors.New("check Google Drive access: user information is empty")
	}

	fmt.Printf("Authenticated as: %s\n", about.User.EmailAddress)

	return nil
}

func (c *Client) ListFiles(ctx context.Context) ([]*drive.File, error) {
	result, err := c.service.Files.List().
		Context(ctx).
		Q("trashed = false").
		Fields("files(id, name, mimeType)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list Google Drive files: %w", err)
	}

	return result.Files, nil
}
