package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type testResponse struct {
	Status int
	Body   []byte
}

type userResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	OrganizationID string    `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
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

type chatMessagePreview struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type chatSummaryResponse struct {
	User        userResponse        `json:"user"`
	LastMessage *chatMessagePreview `json:"lastMessage,omitempty"`
}

type chatsResponse struct {
	Chats []chatSummaryResponse `json:"chats"`
}

type chatMessageResponse struct {
	ID              string    `json:"id"`
	SenderUserID    string    `json:"senderUserId"`
	RecipientUserID string    `json:"recipientUserId"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"createdAt"`
}

type chatMessagesResponse struct {
	Messages []chatMessageResponse `json:"messages"`
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

func TestAuthAndUsersFlow(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	email1 := uniqueEmail("flow1")
	email2 := uniqueEmail("flow2")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email1, email2})

	register1 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email1,
		"password": password,
	}, "")
	if register1.Status != http.StatusCreated {
		t.Fatalf("register1 expected 201, got %d body=%s", register1.Status, string(register1.Body))
	}

	register2 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email2,
		"password": password,
	}, "")
	if register2.Status != http.StatusCreated {
		t.Fatalf("register2 expected 201, got %d body=%s", register2.Status, string(register2.Body))
	}

	var auth1 authResponse
	var auth2 authResponse
	if err := json.Unmarshal(register1.Body, &auth1); err != nil {
		t.Fatalf("unmarshal register1: %v", err)
	}
	if err := json.Unmarshal(register2.Body, &auth2); err != nil {
		t.Fatalf("unmarshal register2: %v", err)
	}
	if auth1.Token == "" || auth2.Token == "" {
		t.Fatalf("expected non-empty JWT tokens")
	}
	if auth1.RefreshToken == "" || auth2.RefreshToken == "" {
		t.Fatalf("expected non-empty refresh tokens")
	}
	if auth1.User.OrganizationID == "" || auth2.User.OrganizationID == "" {
		t.Fatalf("expected organization ids on auth payloads")
	}
	if auth1.User.OrganizationID != auth2.User.OrganizationID {
		t.Fatalf("expected both users in the default organization for current SaaS foundation")
	}

	meResp := doRequest(t, router, http.MethodGet, "/me", nil, auth1.Token)
	if meResp.Status != http.StatusOK {
		t.Fatalf("/me expected 200, got %d body=%s", meResp.Status, string(meResp.Body))
	}
	var meUser meResponse
	if err := json.Unmarshal(meResp.Body, &meUser); err != nil {
		t.Fatalf("unmarshal /me: %v", err)
	}
	if meUser.Email != email1 {
		t.Fatalf("expected /me email %s, got %s", email1, meUser.Email)
	}
	if meUser.OrganizationID != auth1.User.OrganizationID {
		t.Fatalf("expected /me organization id %s, got %s", auth1.User.OrganizationID, meUser.OrganizationID)
	}

	usersList := doRequest(t, router, http.MethodGet, "/users", nil, auth1.Token)
	if usersList.Status != http.StatusOK {
		t.Fatalf("/users expected 200, got %d body=%s", usersList.Status, string(usersList.Body))
	}
	var usersPayload usersResponse
	if err := json.Unmarshal(usersList.Body, &usersPayload); err != nil {
		t.Fatalf("unmarshal /users: %v", err)
	}
	if !containsUser(usersPayload.Users, auth1.User.ID) || !containsUser(usersPayload.Users, auth2.User.ID) {
		t.Fatalf("expected user list to contain both registered users")
	}

	userByID := doRequest(t, router, http.MethodGet, "/users/"+auth2.User.ID, nil, auth1.Token)
	if userByID.Status != http.StatusOK {
		t.Fatalf("/users/:id expected 200, got %d body=%s", userByID.Status, string(userByID.Body))
	}
	var listedUser userResponse
	if err := json.Unmarshal(userByID.Body, &listedUser); err != nil {
		t.Fatalf("unmarshal /users/:id: %v", err)
	}
	if listedUser.Email != email2 {
		t.Fatalf("expected /users/:id email %s, got %s", email2, listedUser.Email)
	}
	if listedUser.OrganizationID != auth1.User.OrganizationID {
		t.Fatalf("expected /users/:id organization id %s, got %s", auth1.User.OrganizationID, listedUser.OrganizationID)
	}

	refresh := doRequest(t, router, http.MethodPost, "/auth/refresh", map[string]string{
		"refreshToken": auth1.RefreshToken,
	}, "")
	if refresh.Status != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d body=%s", refresh.Status, string(refresh.Body))
	}
	var refreshed authResponse
	if err := json.Unmarshal(refresh.Body, &refreshed); err != nil {
		t.Fatalf("unmarshal refresh: %v", err)
	}
	if refreshed.Token == "" || refreshed.RefreshToken == "" {
		t.Fatalf("expected non-empty refreshed tokens")
	}
	if refreshed.RefreshToken == auth1.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}

	logout := doRequest(t, router, http.MethodPost, "/auth/logout", map[string]string{
		"refreshToken": refreshed.RefreshToken,
	}, "")
	if logout.Status != http.StatusNoContent {
		t.Fatalf("logout expected 204, got %d body=%s", logout.Status, string(logout.Body))
	}

	refreshAfterLogout := doRequest(t, router, http.MethodPost, "/auth/refresh", map[string]string{
		"refreshToken": refreshed.RefreshToken,
	}, "")
	if refreshAfterLogout.Status != http.StatusUnauthorized {
		t.Fatalf("refresh after logout expected 401, got %d body=%s", refreshAfterLogout.Status, string(refreshAfterLogout.Body))
	}
}

func TestUnauthorizedEndpoints(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	for _, endpoint := range []string{
		"/me",
		"/users",
		"/users/11111111-1111-1111-1111-111111111111",
		"/chats",
		"/chats/11111111-1111-1111-1111-111111111111/messages",
		"/posts",
		"/posts/11111111-1111-1111-1111-111111111111",
		"/posts/11111111-1111-1111-1111-111111111111/comments",
		"/notifications",
		"/files",
		"/files/11111111-1111-1111-1111-111111111111",
	} {
		resp := doRequest(t, router, http.MethodGet, endpoint, nil, "")
		if resp.Status != http.StatusUnauthorized {
			t.Fatalf("%s expected 401 without token, got %d body=%s", endpoint, resp.Status, string(resp.Body))
		}
	}
}

func TestChatFlow(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	email1 := uniqueEmail("chat1")
	email2 := uniqueEmail("chat2")
	email3 := uniqueEmail("chat3")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email1, email2, email3})

	register1 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email1,
		"password": password,
	}, "")
	register2 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email2,
		"password": password,
	}, "")
	register3 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email3,
		"password": password,
	}, "")

	if register1.Status != http.StatusCreated || register2.Status != http.StatusCreated || register3.Status != http.StatusCreated {
		t.Fatalf("registers expected 201, got [%d, %d, %d]", register1.Status, register2.Status, register3.Status)
	}

	var auth1 authResponse
	var auth2 authResponse
	var auth3 authResponse
	if err := json.Unmarshal(register1.Body, &auth1); err != nil {
		t.Fatalf("unmarshal register1: %v", err)
	}
	if err := json.Unmarshal(register2.Body, &auth2); err != nil {
		t.Fatalf("unmarshal register2: %v", err)
	}
	if err := json.Unmarshal(register3.Body, &auth3); err != nil {
		t.Fatalf("unmarshal register3: %v", err)
	}

	send := doRequest(t, router, http.MethodPost, "/chats/"+auth2.User.ID+"/messages", map[string]string{
		"content": "hello there",
	}, auth1.Token)
	if send.Status != http.StatusCreated {
		t.Fatalf("send message expected 201, got %d body=%s", send.Status, string(send.Body))
	}
	var sent chatMessageResponse
	if err := json.Unmarshal(send.Body, &sent); err != nil {
		t.Fatalf("unmarshal sent message: %v", err)
	}
	if sent.Content != "hello there" {
		t.Fatalf("unexpected message content: %s", sent.Content)
	}

	sendThird := doRequest(t, router, http.MethodPost, "/chats/"+auth3.User.ID+"/messages", map[string]string{
		"content": "hello third user",
	}, auth1.Token)
	if sendThird.Status != http.StatusCreated {
		t.Fatalf("send third user message expected 201, got %d body=%s", sendThird.Status, string(sendThird.Body))
	}

	getMessages := doRequest(t, router, http.MethodGet, "/chats/"+auth2.User.ID+"/messages", nil, auth1.Token)
	if getMessages.Status != http.StatusOK {
		t.Fatalf("get messages expected 200, got %d body=%s", getMessages.Status, string(getMessages.Body))
	}
	var messagesPayload chatMessagesResponse
	if err := json.Unmarshal(getMessages.Body, &messagesPayload); err != nil {
		t.Fatalf("unmarshal get messages: %v", err)
	}
	if len(messagesPayload.Messages) == 0 {
		t.Fatalf("expected at least one chat message")
	}

	getChats := doRequest(t, router, http.MethodGet, "/chats", nil, auth1.Token)
	if getChats.Status != http.StatusOK {
		t.Fatalf("get chats expected 200, got %d body=%s", getChats.Status, string(getChats.Body))
	}
	var chatsPayload chatsResponse
	if err := json.Unmarshal(getChats.Body, &chatsPayload); err != nil {
		t.Fatalf("unmarshal get chats: %v", err)
	}
	if !containsChat(chatsPayload.Chats, auth2.User.ID) || !containsChat(chatsPayload.Chats, auth3.User.ID) {
		t.Fatalf("expected chats list to include both conversation users")
	}

	sendMissingUser := doRequest(t, router, http.MethodPost, "/chats/00000000-0000-0000-0000-000000000000/messages", map[string]string{
		"content": "should fail",
	}, auth1.Token)
	if sendMissingUser.Status != http.StatusNotFound {
		t.Fatalf("send to missing user expected 404, got %d body=%s", sendMissingUser.Status, string(sendMissingUser.Body))
	}
}

func TestUsersSupportsLimitAndOffsetQuery(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	password := "Password123"
	emails := []string{
		uniqueEmail("users_limit_a"),
		uniqueEmail("users_limit_b"),
		uniqueEmail("users_limit_c"),
		uniqueEmail("users_limit_d"),
	}
	defer cleanupUsers(t, pool, emails)

	for _, email := range emails {
		resp := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": password}, "")
		if resp.Status != http.StatusCreated {
			t.Fatalf("register expected 201, got %d body=%s", resp.Status, string(resp.Body))
		}
	}

	login := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": emails[0], "password": password}, "")
	if login.Status != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", login.Status, string(login.Body))
	}
	var auth authResponse
	if err := json.Unmarshal(login.Body, &auth); err != nil {
		t.Fatalf("unmarshal login: %v", err)
	}

	firstPage := doRequest(t, router, http.MethodGet, "/users?limit=1", nil, auth.Token)
	if firstPage.Status != http.StatusOK {
		t.Fatalf("first page /users expected 200, got %d body=%s", firstPage.Status, string(firstPage.Body))
	}
	var firstPayload usersResponse
	if err := json.Unmarshal(firstPage.Body, &firstPayload); err != nil {
		t.Fatalf("unmarshal first page /users: %v", err)
	}
	if len(firstPayload.Users) != 1 {
		t.Fatalf("expected first page to return exactly 1 user, got %d", len(firstPayload.Users))
	}

	secondPage := doRequest(t, router, http.MethodGet, "/users?limit=1&offset=1", nil, auth.Token)
	if secondPage.Status != http.StatusOK {
		t.Fatalf("second page /users expected 200, got %d body=%s", secondPage.Status, string(secondPage.Body))
	}
	var secondPayload usersResponse
	if err := json.Unmarshal(secondPage.Body, &secondPayload); err != nil {
		t.Fatalf("unmarshal second page /users: %v", err)
	}
	if len(secondPayload.Users) != 1 {
		t.Fatalf("expected second page to return exactly 1 user, got %d", len(secondPayload.Users))
	}
	if firstPayload.Users[0].ID == secondPayload.Users[0].ID {
		t.Fatalf("expected offset=1 to return a different user than first page")
	}
}

func TestChatsSupportsLimitAndOffsetQuery(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	password := "Password123"
	emails := []string{
		uniqueEmail("chats_limit_a"),
		uniqueEmail("chats_limit_b"),
		uniqueEmail("chats_limit_c"),
	}
	defer cleanupUsers(t, pool, emails)

	regA := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": emails[0], "password": password}, "")
	regB := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": emails[1], "password": password}, "")
	regC := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": emails[2], "password": password}, "")
	if regA.Status != http.StatusCreated || regB.Status != http.StatusCreated || regC.Status != http.StatusCreated {
		t.Fatalf("registers expected 201, got [%d, %d, %d]", regA.Status, regB.Status, regC.Status)
	}

	var authA authResponse
	var authB authResponse
	var authC authResponse
	if err := json.Unmarshal(regA.Body, &authA); err != nil {
		t.Fatalf("unmarshal authA: %v", err)
	}
	if err := json.Unmarshal(regB.Body, &authB); err != nil {
		t.Fatalf("unmarshal authB: %v", err)
	}
	if err := json.Unmarshal(regC.Body, &authC); err != nil {
		t.Fatalf("unmarshal authC: %v", err)
	}

	sendB := doRequest(t, router, http.MethodPost, "/chats/"+authB.User.ID+"/messages", map[string]string{"content": "hello b"}, authA.Token)
	sendC := doRequest(t, router, http.MethodPost, "/chats/"+authC.User.ID+"/messages", map[string]string{"content": "hello c"}, authA.Token)
	if sendB.Status != http.StatusCreated || sendC.Status != http.StatusCreated {
		t.Fatalf("send messages expected 201, got [%d, %d]", sendB.Status, sendC.Status)
	}

	firstPage := doRequest(t, router, http.MethodGet, "/chats?limit=1", nil, authA.Token)
	if firstPage.Status != http.StatusOK {
		t.Fatalf("first page /chats expected 200, got %d body=%s", firstPage.Status, string(firstPage.Body))
	}
	var firstPayload chatsResponse
	if err := json.Unmarshal(firstPage.Body, &firstPayload); err != nil {
		t.Fatalf("unmarshal first page /chats: %v", err)
	}
	if len(firstPayload.Chats) != 1 {
		t.Fatalf("expected first page to return exactly 1 chat, got %d", len(firstPayload.Chats))
	}

	secondPage := doRequest(t, router, http.MethodGet, "/chats?limit=1&offset=1", nil, authA.Token)
	if secondPage.Status != http.StatusOK {
		t.Fatalf("second page /chats expected 200, got %d body=%s", secondPage.Status, string(secondPage.Body))
	}
	var secondPayload chatsResponse
	if err := json.Unmarshal(secondPage.Body, &secondPayload); err != nil {
		t.Fatalf("unmarshal second page /chats: %v", err)
	}
	if len(secondPayload.Chats) != 1 {
		t.Fatalf("expected second page to return exactly 1 chat, got %d", len(secondPayload.Chats))
	}
	if firstPayload.Chats[0].User.ID == secondPayload.Chats[0].User.ID {
		t.Fatalf("expected offset=1 to return a different chat than first page")
	}
}

func TestChatMessagesSupportsLimitQuery(t *testing.T) {
	router, pool := setupTestRouter(t)
	defer pool.Close()

	email1 := uniqueEmail("chat_limit_1")
	email2 := uniqueEmail("chat_limit_2")
	password := "Password123"
	defer cleanupUsers(t, pool, []string{email1, email2})

	register1 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email1,
		"password": password,
	}, "")
	register2 := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{
		"email":    email2,
		"password": password,
	}, "")
	if register1.Status != http.StatusCreated || register2.Status != http.StatusCreated {
		t.Fatalf("registers expected 201, got [%d, %d]", register1.Status, register2.Status)
	}

	var auth1 authResponse
	var auth2 authResponse
	if err := json.Unmarshal(register1.Body, &auth1); err != nil {
		t.Fatalf("unmarshal register1: %v", err)
	}
	if err := json.Unmarshal(register2.Body, &auth2); err != nil {
		t.Fatalf("unmarshal register2: %v", err)
	}

	send1 := doRequest(t, router, http.MethodPost, "/chats/"+auth2.User.ID+"/messages", map[string]string{"content": "m1"}, auth1.Token)
	send2 := doRequest(t, router, http.MethodPost, "/chats/"+auth2.User.ID+"/messages", map[string]string{"content": "m2"}, auth1.Token)
	if send1.Status != http.StatusCreated || send2.Status != http.StatusCreated {
		t.Fatalf("send messages expected 201, got [%d, %d]", send1.Status, send2.Status)
	}

	getMessages := doRequest(t, router, http.MethodGet, "/chats/"+auth2.User.ID+"/messages?limit=1", nil, auth1.Token)
	if getMessages.Status != http.StatusOK {
		t.Fatalf("get messages with limit expected 200, got %d body=%s", getMessages.Status, string(getMessages.Body))
	}
	var payload chatMessagesResponse
	if err := json.Unmarshal(getMessages.Body, &payload); err != nil {
		t.Fatalf("unmarshal messages payload: %v", err)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("expected exactly 1 message with limit=1, got %d", len(payload.Messages))
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

	a := &app{
		dbPool:    pool,
		jwtSecret: []byte("integration-test-secret"),
	}

	return setupRouter(a), pool
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
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "@boilerplate.test"
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

func containsChat(chats []chatSummaryResponse, userID string) bool {
	for _, chat := range chats {
		if chat.User.ID == userID {
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
