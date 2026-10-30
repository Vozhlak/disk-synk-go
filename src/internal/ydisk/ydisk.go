package ydisk

import (
	"bufio"
	"context"
	"crypto/rand"
	"disk-sync/src/internal/storage"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

const (
	authURL         = "https://oauth.yandex.ru/authorize"
	tokenURL        = "https://oauth.yandex.ru/token"
	diskInfoURL     = "https://cloud-api.yandex.net/v1/disk/"
	diskResourceURL = "https://cloud-api.yandex.net/v1/disk/resources?path=/"

	verificationCodeURL = "https://oauth.yandex.ru/verification_code"

	scopeDiskRead = "cloud_api:disk.read"
)

type Client struct {
	httpClient *http.Client
	token      string
}

type ydiskFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ydiskEmbedded struct {
	Items []ydiskFile `json:"items"`
}

type ydiskResourcesResponse struct {
	Embedded ydiskEmbedded `json:"_embedded"`
}

func New(ctx context.Context, clientID, clientSecret, tokenPath string) (*Client, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("yandex disk: client_id is required")
	}

	if strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("yandex disk: client_secret is required")
	}

	if strings.TrimSpace(tokenPath) == "" {
		return nil, errors.New("yandex disk: token_path is required")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  verificationCodeURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		Scopes: []string{scopeDiskRead},
	}

	token, err := loadOrAuthorizeToken(ctx, cfg, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("yandex disk client: authenticate: %w", err)
	}

	return &Client{
		httpClient: cfg.Client(ctx, token),
		token:      token.AccessToken,
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

func authorize(ctx context.Context, cfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate oauth state: %w", err)
	}

	authUrl := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("force_confirm", "yes"))

	fmt.Println("Go to the following link in your browser:")
	fmt.Println(authUrl)
	fmt.Println()
	fmt.Print("Enter the confirmation code: ")

	code, err := readConfirmationCode()
	if err != nil {
		return nil, fmt.Errorf("read confirmation code: %w", err)
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange confirmation code: %w", err)
	}

	if err = saveToken(tokenPath, token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return token, nil
}

func readConfirmationCode() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	code, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return "", errors.New("confirmation code is empty")
	}

	return code, nil
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, diskInfoURL, nil)
	if err != nil {
		return fmt.Errorf("create Yandex Disk request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("check Yandex Disk access: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf(
			"check Yandex Disk access: unexpected status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	return nil
}

func (c *Client) ListFiles(ctx context.Context) ([]storage.File, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, diskResourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Yandex Disk request: %w", err)
	}

	req.Header.Set("Authorization", "OAuth "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list Yandex Disk files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

		return nil, fmt.Errorf(
			"list Yandex Disk files: unexpected status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var result ydiskResourcesResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Yandex Disk response: %w", err)
	}

	var files []storage.File

	for _, file := range result.Embedded.Items {
		files = append(files, storage.File{
			Name: file.Name,
			Type: file.Type,
		})
	}

	return files, nil
}
