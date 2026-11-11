package httpapi

import (
	"testing"

	"etheyl.local/backend/internal/auth"
	"etheyl.local/backend/internal/store"
)

func TestIntegrationAdminAccess(t *testing.T) {
	h := setup(t)
	h.expect("GET", "/admin/status", "", 401)
	hash, err := auth.HashPassword("administrator password 2026")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.server.q.CreateUser(t.Context(), store.CreateUserParams{
		Email: "admin@example.com", Nickname: "admin", PasswordHash: hash, EmailVerified: true, IsAdmin: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.expect("POST", "/auth/login", `{"email":"admin@example.com","password":"administrator password 2026"}`, 200)
	h.expect("GET", "/admin/status", "", 200)
}
