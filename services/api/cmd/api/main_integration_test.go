package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	authfeature "example.com/api/internal/features/auth"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type testResponse struct {
	Status int
	Body   []byte
}

type userResponse struct {
	ID        string    `json:"id"`
	FirstName string    `json:"firstName"`
	CreatedAt time.Time `json:"createdAt"`
}

type meResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	OrganizationID string    `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type authResponse struct {
	Token        string     `json:"token"`
	AccessToken  string     `json:"accessToken"`
	RefreshToken string     `json:"refreshToken"`
	User         meResponse `json:"user"`
}

type usersResponse struct {
	Users []userResponse `json:"users"`
}

type postResponse struct {
	ID           string    `json:"id"`
	AuthorUserID string    `json:"authorUserId"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type postsResponse struct {
	Posts []postResponse `json:"posts"`
}

type commentResponse struct {
	ID           string    `json:"id"`
	PostID       string    `json:"postId"`
	AuthorUserID string    `json:"authorUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type commentsResponse struct {
	Comments []commentResponse `json:"comments"`
}

type notificationResponse struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	IsRead    bool       `json:"isRead"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type notificationsResponse struct {
	Notifications []notificationResponse `json:"notifications"`
}

type fileResponse struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"ownerUserId"`
	Filename    string    `json:"filename"`
	MimeType    string    `json:"mimeType"`
	SizeBytes   int64     `json:"sizeBytes"`
	StorageKey  string    `json:"storageKey"`
	CreatedAt   time.Time `json:"createdAt"`
}

type filesResponse struct {
	Files []fileResponse `json:"files"`
}

func TestHealthEndpoint(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	resp := doRequest(t, router, http.MethodGet, "/health", nil, "")
	if resp.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Status, string(resp.Body))
	}
	if !strings.Contains(string(resp.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", string(resp.Body))
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Request-ID"); got != "req-integration-fixed-id" {
		t.Fatalf("expected response X-Request-ID to match input, got %q", got)
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	router, pool := setupTestRouter(t)
	pool.Close()

	resp := doRequest(t, router, http.MethodGet, "/health", nil, "")
	if resp.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when DB is unavailable, got %d body=%s", resp.Status, string(resp.Body))
	}
	if !strings.Contains(string(resp.Body), `"status":"unavailable"`) {
		t.Fatalf("unexpected health body when db unavailable: %s", string(resp.Body))
	}
}

func TestAuthFlow(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	email := uniqueEmail("flow")
	defer cleanupUsers(t, pool, []string{email})

	weak := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "onlyletters"}, "")
	if weak.Status != http.StatusBadRequest {
		t.Fatalf("weak password expected 400, got %d body=%s", weak.Status, weak.Body)
	}

	register := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    "  " + strings.ToUpper(email) + " ",
		"password": testPassword,
	}, "")
	if register.Status != http.StatusCreated {
		t.Fatalf("register expected 201, got %d body=%s", register.Status, register.Body)
	}
	var auth authResponse
	mustJSON(t, register.Body, &auth)
	if auth.User.Email != email || auth.AccessToken == "" || auth.RefreshToken == "" {
		t.Fatalf("unexpected register payload: %+v", auth)
	}

	duplicate := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	if duplicate.Status != http.StatusConflict {
		t.Fatalf("duplicate register expected 409, got %d", duplicate.Status)
	}

	badLogin := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": "Wrong12345"}, "")
	if badLogin.Status != http.StatusUnauthorized {
		t.Fatalf("bad login expected 401, got %d", badLogin.Status)
	}
	login := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": email, "password": testPassword}, "")
	if login.Status != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", login.Status, login.Body)
	}
	var loggedIn authResponse
	mustJSON(t, login.Body, &loggedIn)

	me := doRequest(t, router, http.MethodGet, "/me", nil, loggedIn.AccessToken)
	if me.Status != http.StatusOK || !strings.Contains(string(me.Body), email) {
		t.Fatalf("/me expected 200 with own email, got %d body=%s", me.Status, me.Body)
	}

	refresh := doRequest(t, router, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": loggedIn.RefreshToken}, "")
	if refresh.Status != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d body=%s", refresh.Status, refresh.Body)
	}
	var refreshed authResponse
	mustJSON(t, refresh.Body, &refreshed)
	if refreshed.RefreshToken == loggedIn.RefreshToken {
		t.Fatalf("refresh token must rotate")
	}
	reuse := doRequest(t, router, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": loggedIn.RefreshToken}, "")
	if reuse.Status != http.StatusUnauthorized {
		t.Fatalf("rotated refresh token reuse expected 401, got %d", reuse.Status)
	}

	logout := doRequest(t, router, http.MethodPost, "/auth/logout", map[string]string{"refreshToken": refreshed.RefreshToken}, "")
	if logout.Status != http.StatusNoContent {
		t.Fatalf("logout expected 204, got %d", logout.Status)
	}
	// Session revocation is immediate, even though the JWT is not expired yet.
	afterLogout := doRequest(t, router, http.MethodGet, "/me", nil, refreshed.AccessToken)
	if afterLogout.Status != http.StatusUnauthorized {
		t.Fatalf("/me after logout expected 401, got %d", afterLogout.Status)
	}
}

func TestUnauthorizedEndpoints(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/me"},
		{http.MethodGet, "/users"},
		{http.MethodGet, "/chats"},
		{http.MethodGet, "/discovery"},
		{http.MethodPost, "/swipes"},
		{http.MethodGet, "/me/profile"},
		{http.MethodGet, "/notifications"},
		{http.MethodGet, "/events"},
		{http.MethodDelete, "/me"},
	} {
		resp := doRequest(t, router, tc.method, tc.path, nil, "")
		if resp.Status != http.StatusUnauthorized {
			t.Errorf("%s %s without token expected 401, got %d", tc.method, tc.path, resp.Status)
		}
		resp = doRequest(t, router, tc.method, tc.path, nil, "not-a-jwt")
		if resp.Status != http.StatusUnauthorized {
			t.Errorf("%s %s with invalid token expected 401, got %d", tc.method, tc.path, resp.Status)
		}
	}
}

func TestUsersEndpointNeverLeaksEmails(t *testing.T) {
	router, pool := setupTestRouter(t)

	a := registerUser(t, router, pool, "users_a")
	b := registerUser(t, router, pool, "users_b")

	list := doRequest(t, router, http.MethodGet, "/users", nil, a.AccessToken)
	if list.Status != http.StatusOK {
		t.Fatalf("/users expected 200, got %d body=%s", list.Status, list.Body)
	}
	var payload usersResponse
	mustJSON(t, list.Body, &payload)
	if containsUser(payload.Users, b.User.ID) {
		t.Fatalf("/users must only list matched contacts")
	}
	if strings.Contains(string(list.Body), "@") {
		t.Fatalf("/users must never expose e-mail addresses: %s", list.Body)
	}
	if resp := doRequest(t, router, http.MethodGet, "/users/"+b.User.ID, nil, a.AccessToken); resp.Status != http.StatusNotFound {
		t.Fatalf("unmatched /users/:id expected 404, got %d", resp.Status)
	}
	if resp := doRequest(t, router, http.MethodGet, "/users?limit=5&offset=0", nil, a.AccessToken); resp.Status != http.StatusOK {
		t.Fatalf("/users pagination expected 200, got %d", resp.Status)
	}
}

func TestPostsCommentsNotificationsAndFilesFlow(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	email := uniqueEmail("content_flow")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email})

	register := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	if register.Status != http.StatusCreated {
		t.Fatalf("register expected 201, got %d body=%s", register.Status, string(register.Body))
	}

	var auth authResponse
	if err := json.Unmarshal(register.Body, &auth); err != nil {
		t.Fatalf("unmarshal register: %v", err)
	}

	createPost := doRequest(t, router, http.MethodPost, "/posts", map[string]string{
		"title": "First boilerplate post",
		"body":  "This is a generic post body.",
	}, auth.Token)
	if createPost.Status != http.StatusCreated {
		t.Fatalf("create post expected 201, got %d body=%s", createPost.Status, string(createPost.Body))
	}

	var createdPost postResponse
	if err := json.Unmarshal(createPost.Body, &createdPost); err != nil {
		t.Fatalf("unmarshal create post: %v", err)
	}
	if createdPost.ID == "" || createdPost.AuthorUserID != auth.User.ID {
		t.Fatalf("unexpected created post payload: %+v", createdPost)
	}

	listPosts := doRequest(t, router, http.MethodGet, "/posts", nil, auth.Token)
	if listPosts.Status != http.StatusOK {
		t.Fatalf("list posts expected 200, got %d body=%s", listPosts.Status, string(listPosts.Body))
	}
	var postsPayload postsResponse
	if err := json.Unmarshal(listPosts.Body, &postsPayload); err != nil {
		t.Fatalf("unmarshal list posts: %v", err)
	}
	if !containsPost(postsPayload.Posts, createdPost.ID) {
		t.Fatalf("expected posts list to contain created post")
	}

	getPost := doRequest(t, router, http.MethodGet, "/posts/"+createdPost.ID, nil, auth.Token)
	if getPost.Status != http.StatusOK {
		t.Fatalf("get post expected 200, got %d body=%s", getPost.Status, string(getPost.Body))
	}
	var fetchedPost postResponse
	if err := json.Unmarshal(getPost.Body, &fetchedPost); err != nil {
		t.Fatalf("unmarshal get post: %v", err)
	}
	if fetchedPost.ID != createdPost.ID {
		t.Fatalf("expected fetched post id %s, got %s", createdPost.ID, fetchedPost.ID)
	}

	createComment := doRequest(t, router, http.MethodPost, "/posts/"+createdPost.ID+"/comments", map[string]string{
		"content": "First comment on the post",
	}, auth.Token)
	if createComment.Status != http.StatusCreated {
		t.Fatalf("create comment expected 201, got %d body=%s", createComment.Status, string(createComment.Body))
	}

	var createdComment commentResponse
	if err := json.Unmarshal(createComment.Body, &createdComment); err != nil {
		t.Fatalf("unmarshal create comment: %v", err)
	}
	if createdComment.PostID != createdPost.ID {
		t.Fatalf("expected comment post id %s, got %s", createdPost.ID, createdComment.PostID)
	}

	listComments := doRequest(t, router, http.MethodGet, "/posts/"+createdPost.ID+"/comments", nil, auth.Token)
	if listComments.Status != http.StatusOK {
		t.Fatalf("list comments expected 200, got %d body=%s", listComments.Status, string(listComments.Body))
	}
	var commentsPayload commentsResponse
	if err := json.Unmarshal(listComments.Body, &commentsPayload); err != nil {
		t.Fatalf("unmarshal list comments: %v", err)
	}
	if !containsComment(commentsPayload.Comments, createdComment.ID) {
		t.Fatalf("expected comments list to contain created comment")
	}

	createNotification := doRequest(t, router, http.MethodPost, "/notifications", map[string]string{
		"type":  "system",
		"title": "Welcome",
		"body":  "Your workspace is ready.",
	}, auth.Token)
	if createNotification.Status != http.StatusCreated {
		t.Fatalf("create notification expected 201, got %d body=%s", createNotification.Status, string(createNotification.Body))
	}

	var createdNotification notificationResponse
	if err := json.Unmarshal(createNotification.Body, &createdNotification); err != nil {
		t.Fatalf("unmarshal create notification: %v", err)
	}
	if createdNotification.UserID != auth.User.ID || createdNotification.IsRead {
		t.Fatalf("unexpected notification payload: %+v", createdNotification)
	}

	listNotifications := doRequest(t, router, http.MethodGet, "/notifications", nil, auth.Token)
	if listNotifications.Status != http.StatusOK {
		t.Fatalf("list notifications expected 200, got %d body=%s", listNotifications.Status, string(listNotifications.Body))
	}
	var notificationsPayload notificationsResponse
	if err := json.Unmarshal(listNotifications.Body, &notificationsPayload); err != nil {
		t.Fatalf("unmarshal list notifications: %v", err)
	}
	if !containsNotification(notificationsPayload.Notifications, createdNotification.ID) {
		t.Fatalf("expected notifications list to contain created notification")
	}

	markRead := doRequest(t, router, http.MethodPost, "/notifications/"+createdNotification.ID+"/read", nil, auth.Token)
	if markRead.Status != http.StatusOK {
		t.Fatalf("mark notification read expected 200, got %d body=%s", markRead.Status, string(markRead.Body))
	}
	var readNotification notificationResponse
	if err := json.Unmarshal(markRead.Body, &readNotification); err != nil {
		t.Fatalf("unmarshal mark read: %v", err)
	}
	if !readNotification.IsRead || readNotification.ReadAt == nil {
		t.Fatalf("expected notification to be marked as read, got %+v", readNotification)
	}

	createFile := doRequest(t, router, http.MethodPost, "/files", map[string]any{
		"filename":   "document.txt",
		"mimeType":   "text/plain",
		"sizeBytes":  128,
		"storageKey": "uploads/document.txt",
	}, auth.Token)
	if createFile.Status != http.StatusCreated {
		t.Fatalf("create file expected 201, got %d body=%s", createFile.Status, string(createFile.Body))
	}

	var createdFile fileResponse
	if err := json.Unmarshal(createFile.Body, &createdFile); err != nil {
		t.Fatalf("unmarshal create file: %v", err)
	}
	if createdFile.OwnerUserID != auth.User.ID || createdFile.ID == "" {
		t.Fatalf("unexpected file payload: %+v", createdFile)
	}

	listFiles := doRequest(t, router, http.MethodGet, "/files", nil, auth.Token)
	if listFiles.Status != http.StatusOK {
		t.Fatalf("list files expected 200, got %d body=%s", listFiles.Status, string(listFiles.Body))
	}
	var filesPayload filesResponse
	if err := json.Unmarshal(listFiles.Body, &filesPayload); err != nil {
		t.Fatalf("unmarshal list files: %v", err)
	}
	if !containsFile(filesPayload.Files, createdFile.ID) {
		t.Fatalf("expected files list to contain created file")
	}

	getFile := doRequest(t, router, http.MethodGet, "/files/"+createdFile.ID, nil, auth.Token)
	if getFile.Status != http.StatusOK {
		t.Fatalf("get file expected 200, got %d body=%s", getFile.Status, string(getFile.Body))
	}
	var fetchedFile fileResponse
	if err := json.Unmarshal(getFile.Body, &fetchedFile); err != nil {
		t.Fatalf("unmarshal get file: %v", err)
	}
	if fetchedFile.ID != createdFile.ID || fetchedFile.StorageKey != createdFile.StorageKey {
		t.Fatalf("unexpected fetched file payload: %+v", fetchedFile)
	}
}

func TestTenantIsolationAcrossUsersPostsAndChats(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	defaultEmail := uniqueEmail("tenant_default")
	otherEmail := uniqueEmail("tenant_other")
	password := "Password123"

	otherOrgID := createOrganization(t, pool, "tenant-isolation", "Tenant Isolation Workspace")
	defer cleanupOrganizations(t, pool, []string{otherOrgID})
	defer cleanupUsers(t, pool, []string{defaultEmail, otherEmail})

	registerDefault := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    defaultEmail,
		"password": password,
	}, "")
	registerOther := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    otherEmail,
		"password": password,
	}, "")
	if registerDefault.Status != http.StatusCreated || registerOther.Status != http.StatusCreated {
		t.Fatalf("registers expected 201, got [%d, %d]", registerDefault.Status, registerOther.Status)
	}

	var defaultAuth authResponse
	var otherAuth authResponse
	if err := json.Unmarshal(registerDefault.Body, &defaultAuth); err != nil {
		t.Fatalf("unmarshal default auth: %v", err)
	}
	if err := json.Unmarshal(registerOther.Body, &otherAuth); err != nil {
		t.Fatalf("unmarshal other auth: %v", err)
	}

	moveUserToOrganization(t, pool, otherAuth.User.ID, otherOrgID)

	loginOther := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    otherEmail,
		"password": password,
	}, "")
	if loginOther.Status != http.StatusOK {
		t.Fatalf("other org login expected 200, got %d body=%s", loginOther.Status, string(loginOther.Body))
	}
	if err := json.Unmarshal(loginOther.Body, &otherAuth); err != nil {
		t.Fatalf("unmarshal other login auth: %v", err)
	}
	if otherAuth.User.OrganizationID != otherOrgID {
		t.Fatalf("expected moved user organization %s, got %s", otherOrgID, otherAuth.User.OrganizationID)
	}

	createForeignPost := doRequest(t, router, http.MethodPost, "/posts", map[string]string{
		"title": "Other org post",
		"body":  "This should stay inside its tenant.",
	}, otherAuth.Token)
	if createForeignPost.Status != http.StatusCreated {
		t.Fatalf("other org create post expected 201, got %d body=%s", createForeignPost.Status, string(createForeignPost.Body))
	}
	var foreignPost postResponse
	if err := json.Unmarshal(createForeignPost.Body, &foreignPost); err != nil {
		t.Fatalf("unmarshal foreign post: %v", err)
	}

	usersList := doRequest(t, router, http.MethodGet, "/users", nil, defaultAuth.Token)
	if usersList.Status != http.StatusOK {
		t.Fatalf("/users expected 200, got %d body=%s", usersList.Status, string(usersList.Body))
	}
	var usersPayload usersResponse
	if err := json.Unmarshal(usersList.Body, &usersPayload); err != nil {
		t.Fatalf("unmarshal /users payload: %v", err)
	}
	if containsUser(usersPayload.Users, otherAuth.User.ID) {
		t.Fatalf("expected cross-tenant user to be hidden from /users")
	}

	userByID := doRequest(t, router, http.MethodGet, "/users/"+otherAuth.User.ID, nil, defaultAuth.Token)
	if userByID.Status != http.StatusNotFound {
		t.Fatalf("cross-tenant /users/:id expected 404, got %d body=%s", userByID.Status, string(userByID.Body))
	}

	listPosts := doRequest(t, router, http.MethodGet, "/posts", nil, defaultAuth.Token)
	if listPosts.Status != http.StatusOK {
		t.Fatalf("/posts expected 200, got %d body=%s", listPosts.Status, string(listPosts.Body))
	}
	var postsPayload postsResponse
	if err := json.Unmarshal(listPosts.Body, &postsPayload); err != nil {
		t.Fatalf("unmarshal /posts payload: %v", err)
	}
	if containsPost(postsPayload.Posts, foreignPost.ID) {
		t.Fatalf("expected cross-tenant post to be hidden from /posts")
	}

	getForeignPost := doRequest(t, router, http.MethodGet, "/posts/"+foreignPost.ID, nil, defaultAuth.Token)
	if getForeignPost.Status != http.StatusNotFound {
		t.Fatalf("cross-tenant /posts/:id expected 404, got %d body=%s", getForeignPost.Status, string(getForeignPost.Body))
	}

	listForeignComments := doRequest(t, router, http.MethodGet, "/posts/"+foreignPost.ID+"/comments", nil, defaultAuth.Token)
	if listForeignComments.Status != http.StatusNotFound {
		t.Fatalf("cross-tenant /posts/:id/comments expected 404, got %d body=%s", listForeignComments.Status, string(listForeignComments.Body))
	}

	sendCrossTenantMessage := doRequest(t, router, http.MethodPost, "/chats/"+otherAuth.User.ID+"/messages", map[string]string{
		"content": "should not cross tenants",
	}, defaultAuth.Token)
	if sendCrossTenantMessage.Status != http.StatusNotFound {
		t.Fatalf("cross-tenant chat send expected 404, got %d body=%s", sendCrossTenantMessage.Status, string(sendCrossTenantMessage.Body))
	}
}

const testPassword = "Password123"

// testEnv exposes the collaborators tests need to observe side effects.
type testEnv struct {
	mail *mailer.Memory
	hub  *realtime.Hub
}

var lastTestEnv testEnv

func setupTestRouter(t *testing.T) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}

	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}

	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	// Registered first so it runs last, after per-test data cleanups.
	t.Cleanup(pool.Close)

	storage, err := media.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	env := testEnv{mail: &mailer.Memory{}, hub: realtime.NewHub()}
	lastTestEnv = env
	generous := func() *middleware.RateLimiter { return middleware.NewRateLimiter(10_000, time.Minute, 10_000) }

	a := &app{
		dbPool:    pool,
		jwtSecret: []byte("integration-test-secret-at-least-32-chars"),
		storage:   storage,
		mailer:    env.mail,
		hub:       env.hub,
		limits: limits{
			auth:     authfeature.Limiters{Credentials: generous(), Recovery: generous()},
			swipes:   generous(),
			messages: generous(),
			reports:  generous(),
			uploads:  generous(),
		},
	}

	return setupRouter(a), pool
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func mustJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("unmarshal %s: %v", body, err)
	}
}

// registerUser creates an account and schedules its deletion.
func registerUser(t *testing.T, router http.Handler, pool *pgxpool.Pool, prefix string) authResponse {
	t.Helper()
	email := uniqueEmail(prefix)
	t.Cleanup(func() { cleanupUsers(t, pool, []string{email}) })
	resp := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": testPassword}, "")
	if resp.Status != http.StatusCreated {
		t.Fatalf("register %s expected 201, got %d body=%s", prefix, resp.Status, resp.Body)
	}
	var auth authResponse
	mustJSON(t, resp.Body, &auth)
	return auth
}

func doRequest(t *testing.T, handler http.Handler, method, path string, payload any, token string) testResponse {
	t.Helper()

	var bodyBytes []byte
	var err error
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

func uniqueEmail(prefix string) string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "_" + hex.EncodeToString(b[:]) + "@lueur.test"
}

func cleanupUsers(t *testing.T, pool *pgxpool.Pool, emails []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, emails)
	if err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}

func createOrganization(t *testing.T, pool *pgxpool.Pool, slug, name string) string {
	t.Helper()
	var organizationID string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO organizations (slug, name)
		VALUES ($1, $2)
		RETURNING id
	`, slug, name).Scan(&organizationID)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return organizationID
}

func moveUserToOrganization(t *testing.T, pool *pgxpool.Pool, userID, organizationID string) {
	t.Helper()
	commandTag, err := pool.Exec(context.Background(), `
		UPDATE users
		SET organization_id = $2
		WHERE id = $1
	`, userID, organizationID)
	if err != nil {
		t.Fatalf("move user to organization: %v", err)
	}
	if commandTag.RowsAffected() != 1 {
		t.Fatalf("expected 1 moved user, got %d", commandTag.RowsAffected())
	}
}

func cleanupOrganizations(t *testing.T, pool *pgxpool.Pool, organizationIDs []string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `DELETE FROM organizations WHERE id = ANY($1)`, organizationIDs)
	if err != nil {
		t.Fatalf("cleanup organizations: %v", err)
	}
}

func containsUser(users []userResponse, userID string) bool {
	for _, user := range users {
		if user.ID == userID {
			return true
		}
	}
	return false
}

func containsPost(posts []postResponse, postID string) bool {
	for _, post := range posts {
		if post.ID == postID {
			return true
		}
	}
	return false
}

func containsComment(comments []commentResponse, commentID string) bool {
	for _, comment := range comments {
		if comment.ID == commentID {
			return true
		}
	}
	return false
}

func containsNotification(notifications []notificationResponse, notificationID string) bool {
	for _, notification := range notifications {
		if notification.ID == notificationID {
			return true
		}
	}
	return false
}

func containsFile(files []fileResponse, fileID string) bool {
	for _, file := range files {
		if file.ID == fileID {
			return true
		}
	}
	return false
}
