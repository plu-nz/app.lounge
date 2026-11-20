package stremio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Stremio account endpoints, as used by stremio-core. Tests override them.
var (
	apiURL  = "https://api.strem.io/api/"
	linkURL = "https://link.stremio.com/api/v2/"
)

// Account is a signed-in Stremio account.
type Account struct {
	AuthKey string `json:"authKey"`
	Email   string `json:"email"`
}

// LinkCode is a pairing code the user confirms on another device.
type LinkCode struct {
	Code   string `json:"code"`
	Link   string `json:"link"`
	QRCode string `json:"qrcode"`
}

// Descriptor is an addon in the account's collection.
type Descriptor struct {
	TransportURL string          `json:"transportUrl"`
	Manifest     json.RawMessage `json:"manifest"`
	Flags        struct {
		Official  bool `json:"official"`
		Protected bool `json:"protected"`
	} `json:"flags"`
}

// APIError is an error returned by the Stremio API.
type APIError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *APIError) Error() string { return e.Message }

type apiUser struct {
	Email string `json:"email"`
}

// Login signs in with email and password.
func (c *Client) Login(ctx context.Context, email, password string) (Account, error) {
	var res struct {
		AuthKey string  `json:"authKey"`
		User    apiUser `json:"user"`
	}
	err := c.api(ctx, "login", map[string]any{"type": "Login", "email": email, "password": password, "facebook": false}, &res)
	if err != nil {
		return Account{}, err
	}
	if res.AuthKey == "" {
		return Account{}, errors.New("the Stremio API returned no auth key")
	}
	return Account{AuthKey: res.AuthKey, Email: res.User.Email}, nil
}

// User returns the account for an auth key, to show who is signed in.
func (c *Client) User(ctx context.Context, authKey string) (Account, error) {
	var res apiUser
	if err := c.api(ctx, "getUser", map[string]any{"type": "GetUser", "authKey": authKey}, &res); err != nil {
		return Account{}, err
	}
	return Account{AuthKey: authKey, Email: res.Email}, nil
}

// Logout ends the session on the server.
func (c *Client) Logout(ctx context.Context, authKey string) error {
	var res json.RawMessage
	return c.api(ctx, "logout", map[string]any{"type": "Logout", "authKey": authKey}, &res)
}

// AddonCollection returns the account's installed addons in order.
func (c *Client) AddonCollection(ctx context.Context, authKey string) ([]Descriptor, error) {
	var res struct {
		Addons []Descriptor `json:"addons"`
	}
	err := c.api(ctx, "addonCollectionGet", map[string]any{"type": "AddonCollectionGet", "authKey": authKey, "update": true}, &res)
	return res.Addons, err
}

// CreateLink starts the code/QR sign-in.
func (c *Client) CreateLink(ctx context.Context) (LinkCode, error) {
	var res LinkCode
	err := c.link(ctx, "create", url.Values{"type": {"Create"}}, &res)
	return res, err
}

// ErrLinkPending means the pairing code has not been confirmed yet.
var ErrLinkPending = errors.New("the code has not been confirmed yet")

// ReadLink returns the auth key once the user confirms the code. Until
// then the server answers with an error, reported as ErrLinkPending.
func (c *Client) ReadLink(ctx context.Context, code string) (string, error) {
	var res struct {
		AuthKey string `json:"authKey"`
	}
	err := c.link(ctx, "read", url.Values{"type": {"Read"}, "code": {code}}, &res)
	var apiErr *APIError
	if errors.As(err, &apiErr) || (err == nil && res.AuthKey == "") {
		return "", ErrLinkPending
	}
	return res.AuthKey, err
}

func (c *Client) api(ctx context.Context, method string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+method, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.result(req, out)
}

func (c *Client) link(ctx context.Context, method string, query url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, linkURL+method+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	return c.result(req, out)
}

// result decodes the {"result": ...} or {"error": ...} envelope.
func (c *Client) result(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	var env struct {
		Result json.RawMessage `json:"result"`
		Error  *APIError       `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("stremio api: %s: %s", resp.Status, strings.TrimSpace(string(raw[:min(len(raw), 200)])))
	}
	if env.Error != nil {
		return env.Error
	}
	if len(env.Result) == 0 {
		return fmt.Errorf("stremio api: empty result (%s)", resp.Status)
	}
	return json.Unmarshal(env.Result, out)
}
