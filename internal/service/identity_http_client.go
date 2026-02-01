package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IdentityHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewIdentityHTTPClient(baseURL string, httpClient *http.Client) *IdentityHTTPClient {
	return &IdentityHTTPClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

type identityMeResponse struct {
	User  identityUser   `json:"user"`
	Roles []identityRole `json:"roles"`
}

type identityUser struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type identityRole struct {
	Name    string     `json:"name"`
	ClassID *uuid.UUID `json:"class_id"`
}

func (c *IdentityHTTPClient) GetMe(ctx context.Context, userID uuid.UUID) (IdentityUser, error) {
	if c.baseURL == "" {
		return IdentityUser{}, ErrInvalidInput
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/me", nil)
	if err != nil {
		return IdentityUser{}, err
	}
	req.Header.Set("X-User-ID", userID.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return IdentityUser{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusNotFound:
		return IdentityUser{}, ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return IdentityUser{}, ErrUnauthorized
	default:
		return IdentityUser{}, fmt.Errorf("identity service unexpected status: %d", resp.StatusCode)
	}

	var body identityMeResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&body); err != nil {
		return IdentityUser{}, err
	}

	roles := make([]IdentityRole, 0, len(body.Roles))
	for _, role := range body.Roles {
		roles = append(roles, IdentityRole{Name: role.Name, ClassID: role.ClassID})
	}

	if body.User.ID == uuid.Nil {
		return IdentityUser{}, errors.New("identity response missing id")
	}

	return IdentityUser{ID: body.User.ID, Roles: roles}, nil
}

func DefaultIdentityHTTPClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second}
}
