package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuthServiceRegisterLoginMe(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	authRepo := NewPGRepository(pool)
	svc := NewService(authRepo, []byte("services-test-secret"))
	email := uniqueEmail("authsvc")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email})

	tokens, user, err := svc.Register(context.Background(), "  "+strings.ToUpper(email)+"  ", password, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected non-empty access and refresh tokens")
	}
	if user.Email != email {
		t.Fatalf("expected normalized email %q, got %q", email, user.Email)
	}
	if user.OrganizationID == "" {
		t.Fatalf("expected organization id on registered user")
	}

	_, _, err = svc.Register(context.Background(), email, password, "", "")
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}

	_, _, err = svc.Login(context.Background(), email, "wrong-password1", "", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	loginTokens, loggedIn, err := svc.Login(context.Background(), email, password, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginTokens.AccessToken == "" || loginTokens.RefreshToken == "" {
		t.Fatalf("expected non-empty login tokens")
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("expected login user id %s, got %s", user.ID, loggedIn.ID)
	}
	if loggedIn.OrganizationID != user.OrganizationID {
		t.Fatalf("expected login user organization id %s, got %s", user.OrganizationID, loggedIn.OrganizationID)
	}

	refreshedTokens, refreshedUser, err := svc.Refresh(context.Background(), loginTokens.RefreshToken, "auth-test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshedTokens.AccessToken == "" || refreshedTokens.RefreshToken == "" {
		t.Fatalf("expected non-empty refreshed tokens")
	}
	if refreshedTokens.RefreshToken == loginTokens.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}
	if refreshedUser.OrganizationID != user.OrganizationID {
		t.Fatalf("expected refreshed user organization id %s, got %s", user.OrganizationID, refreshedUser.OrganizationID)
	}

	if err := svc.Logout(context.Background(), refreshedTokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}

	_, _, err = svc.Refresh(context.Background(), refreshedTokens.RefreshToken, "auth-test-agent", "127.0.0.1")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken after logout, got %v", err)
	}

	meUser, err := svc.Me(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("me existing user: %v", err)
	}
	if meUser.Email != email {
		t.Fatalf("expected me email %q, got %q", email, meUser.Email)
	}
	if meUser.OrganizationID != user.OrganizationID {
		t.Fatalf("expected me organization id %s, got %s", user.OrganizationID, meUser.OrganizationID)
	}

	_, err = svc.Me(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUsersServiceListAndGetByID(t *testing.T) {
	pool := setupFeaturesTestDB(t)
	defer pool.Close()

	authRepo := NewPGRepository(pool)
	authSvc := NewService(authRepo, []byte("services-test-secret"))
	usersRepo := usersfeature.NewPGRepository(pool)
	usersSvc := usersfeature.NewService(usersRepo)

	email1 := uniqueEmail("users1")
	email2 := uniqueEmail("users2")
	email3 := uniqueEmail("users3")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email1, email2, email3})

	_, user1, err := authSvc.Register(context.Background(), email1, password, "", "")
	if err != nil {
		t.Fatalf("register user1: %v", err)
	}
	_, user2, err := authSvc.Register(context.Background(), email2, password, "", "")
	if err != nil {
		t.Fatalf("register user2: %v", err)
	}
	_, user3, err := authSvc.Register(context.Background(), email3, password, "", "")
	if err != nil {
		t.Fatalf("register user3: %v", err)
	}

	listed, err := usersSvc.ListWithPagination(context.Background(), user1.OrganizationID, user1.ID, 10, 0)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if !containsUser(listed, user1.ID) || !containsUser(listed, user2.ID) || !containsUser(listed, user3.ID) {
		t.Fatalf("expected user list to contain all registered users")
	}

	gotUser, err := usersSvc.GetByID(context.Background(), user1.OrganizationID, user1.ID, user2.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if gotUser.ID != user2.ID {
		t.Fatalf("expected user id %q, got %q", user2.ID, gotUser.ID)
	}
	if gotUser.OrganizationID != user1.OrganizationID {
		t.Fatalf("expected user organization id %s, got %s", user1.OrganizationID, gotUser.OrganizationID)
	}

	_, err = usersSvc.GetByID(context.Background(), user1.OrganizationID, user1.ID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, usersfeature.ErrUserNotFound) {
		t.Fatalf("expected users ErrUserNotFound, got %v", err)
	}
}

func setupFeaturesTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip features integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}

	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip features integration tests: postgres unavailable (%v)", err)
	}

	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}

	return pool
}

func uniqueEmail(prefix string) string {
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "@boilerplate.test"
}

func cleanupUsers(t *testing.T, pool *pgxpool.Pool, emails []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, emails)
	if err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}

func containsUser(users []usersfeature.User, userID string) bool {
	for _, user := range users {
		if user.ID == userID {
			return true
		}
	}
	return false
}
