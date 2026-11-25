package controller

import (
	"bytes"
	"e-commerce/config"
	entities "e-commerce/core/entities/user"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeUserRepo struct {
	user *entities.User
}

func (f *fakeUserRepo) CreateUser(entities.User) error                 { return nil }
func (f *fakeUserRepo) ReadUserById(string) (*entities.User, error)    { return f.user, nil }
func (f *fakeUserRepo) DeleteUserById(string) error                    { return nil }
func (f *fakeUserRepo) ReadAllUsers() ([]entities.User, error)         { return nil, nil }
func (f *fakeUserRepo) ReadUserByEmail(string) (*entities.User, error) { return f.user, nil }
func (f *fakeUserRepo) UpdateUser(entities.User, string) error         { return nil }
func (f *fakeUserRepo) UpdatePasswordUser(entities.User, string) error { return nil }

type fakeHashService struct{}

func (f *fakeHashService) CompareHash(string, string) error { return nil }
func (f *fakeHashService) HashGenerate(string) (string, error) {
	return "hashed", nil
}

func findTokenCookie(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == "token" {
			return c
		}
	}
	t.Fatalf("expected token cookie in Set-Cookie headers: %v", w.Header().Values("Set-Cookie"))
	return nil
}

func TestAuthController_CookieSameSitePolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origSecret := config.SecretKey
	origCookieSecure := config.CookieSecure
	origCookieDomain := config.CookieDomain
	t.Cleanup(func() {
		config.SecretKey = origSecret
		config.CookieSecure = origCookieSecure
		config.CookieDomain = origCookieDomain
	})

	config.SecretKey = strings.Repeat("a", 32)
	config.CookieDomain = ""

	repo := &fakeUserRepo{
		user: &entities.User{
			ID:         uuid.New(),
			Email:      "user@example.com",
			Password:   "hashed",
			Role:       "user",
			IsVerified: true,
		},
	}
	authCtrl := NewAuthController(repo, &fakeHashService{})

	r := gin.New()
	r.POST("/api/v1/login", authCtrl.Login)
	r.POST("/api/v1/logout", authCtrl.Logout)

	cases := []struct {
		name         string
		cookieSecure bool
		wantSameSite http.SameSite
		wantSecure   bool
	}{
		{
			name:         "Production HTTPS uses SameSite=None and Secure=true",
			cookieSecure: true,
			wantSameSite: http.SameSiteNoneMode,
			wantSecure:   true,
		},
		{
			name:         "Local HTTP uses SameSite=Lax and Secure=false",
			cookieSecure: false,
			wantSameSite: http.SameSiteLaxMode,
			wantSecure:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config.CookieSecure = tc.cookieSecure

			// Login
			loginBody := `{"email":"user@example.com","password":"secretpassword"}`
			loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(loginBody))
			loginReq.Header.Set("Content-Type", "application/json")
			loginW := httptest.NewRecorder()
			r.ServeHTTP(loginW, loginReq)

			if loginW.Code != http.StatusOK {
				t.Fatalf("login expected 200, got %d (body=%s)", loginW.Code, loginW.Body.String())
			}
			loginCookie := findTokenCookie(t, loginW)
			if loginCookie.SameSite != tc.wantSameSite {
				t.Fatalf("login SameSite: got %v, want %v", loginCookie.SameSite, tc.wantSameSite)
			}
			if loginCookie.Secure != tc.wantSecure {
				t.Fatalf("login Secure: got %v, want %v", loginCookie.Secure, tc.wantSecure)
			}
			if !loginCookie.HttpOnly {
				t.Fatalf("login cookie must be HttpOnly")
			}

			// Logout
			logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
			logoutReq.AddCookie(loginCookie)
			logoutW := httptest.NewRecorder()
			r.ServeHTTP(logoutW, logoutReq)

			if logoutW.Code != http.StatusOK {
				t.Fatalf("logout expected 200, got %d (body=%s)", logoutW.Code, logoutW.Body.String())
			}
			logoutCookie := findTokenCookie(t, logoutW)
			if logoutCookie.SameSite != tc.wantSameSite {
				t.Fatalf("logout SameSite: got %v, want %v", logoutCookie.SameSite, tc.wantSameSite)
			}
			if logoutCookie.Secure != tc.wantSecure {
				t.Fatalf("logout Secure: got %v, want %v", logoutCookie.Secure, tc.wantSecure)
			}
			if !logoutCookie.HttpOnly {
				t.Fatalf("logout cookie must be HttpOnly")
			}
			if logoutCookie.MaxAge >= 0 {
				t.Fatalf("logout cookie expected negative MaxAge, got %d", logoutCookie.MaxAge)
			}
		})
	}
}
