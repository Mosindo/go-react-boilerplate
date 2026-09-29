package account_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"example.com/api/internal/features/account"
	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	pool *pgxpool.Pool
	d    deps.Common
	r    *gin.Engine
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testutil.Pool(t)
	d := testutil.Deps(t, nil, nil)
	r := testutil.Router(func(r gin.IRouter) {
		auth.RegisterRoutes(r, pool, d)
		account.RegisterRoutes(r, pool, d)
	})
	return fixture{pool: pool, d: d, r: r}
}

type authResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID string `json:"id"`
	} `json:"user"`
}

func (f fixture) register(t *testing.T, pw string) (authResp, string) {
	t.Helper()
	email := fmt.Sprintf("acct_%d@test.local", time.Now().UnixNano())
	res := testutil.Do(t, f.r, "POST", "/auth/register", map[string]string{"email": email, "password": pw, "birthDate": "1990-05-05"}, "")
	if res.Status != 201 {
		t.Fatalf("register %d %s", res.Status, res.Body)
	}
	var a authResp
	res.JSON(t, &a)
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, a.User.ID) })
	return a, email
}

func TestMeShape(t *testing.T) {
	f := setup(t)
	a, email := f.register(t, "Password!123")

	var me struct {
		ID              string `json:"id"`
		Email           string `json:"email"`
		BirthDate       string `json:"birthDate"`
		Age             int    `json:"age"`
		CreatedAt       string `json:"createdAt"`
		ProfileComplete bool   `json:"profileComplete"`
	}
	res := testutil.Do(t, f.r, "GET", "/me", nil, a.AccessToken)
	if res.Status != 200 {
		t.Fatalf("me %d %s", res.Status, res.Body)
	}
	res.JSON(t, &me)
	if me.ID != a.User.ID || me.Email != email || me.BirthDate != "1990-05-05" || me.Age < 35 || me.ProfileComplete {
		t.Fatalf("unexpected me %+v", me)
	}
	if testutil.Do(t, f.r, "GET", "/me", nil, "").Status != 401 {
		t.Fatal("me without token must be 401")
	}

	ctx := context.Background()
	must := func(sql string, args ...any) {
		if _, err := f.pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	complete := func() bool {
		var m struct {
			ProfileComplete bool `json:"profileComplete"`
		}
		testutil.Do(t, f.r, "GET", "/me", nil, a.AccessToken).JSON(t, &m)
		return m.ProfileComplete
	}
	must(`INSERT INTO profiles (user_id, first_name, gender) VALUES ($1,'Ann','woman')`, a.User.ID)
	if complete() {
		t.Fatal("profile alone must not be complete")
	}
	must(`INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,0,10,10,10)`, a.User.ID, "photos/"+a.User.ID+"/x.jpg")
	if complete() {
		t.Fatal("profile+photo without location must not be complete")
	}
	must(`UPDATE profiles SET latitude=48.85, longitude=2.35, location_updated_at=NOW() WHERE user_id=$1`, a.User.ID)
	if !complete() {
		t.Fatal("expected complete profile")
	}
}

func TestChangePassword(t *testing.T) {
	f := setup(t)
	a, email := f.register(t, "Password!123")
	res := testutil.Do(t, f.r, "POST", "/auth/login", map[string]string{"email": email, "password": "Password!123"}, "")
	var other authResp
	res.JSON(t, &other)

	pw := func(cur, next string) int {
		return testutil.Do(t, f.r, "POST", "/me/password", map[string]string{"currentPassword": cur, "newPassword": next}, a.AccessToken).Status
	}
	if s := pw("wrong-password", "NewPassword!456"); s != 403 {
		t.Fatalf("wrong current want 403 got %d", s)
	}
	if s := pw("Password!123", "short"); s != 400 {
		t.Fatalf("weak want 400 got %d", s)
	}
	if s := pw("Password!123", "NewPassword!456"); s != 204 {
		t.Fatalf("change want 204 got %d", s)
	}
	// Other session revoked, current kept.
	if s := testutil.Do(t, f.r, "POST", "/auth/refresh", map[string]string{"refreshToken": other.RefreshToken}, "").Status; s != 401 {
		t.Fatalf("other session must be revoked, got %d", s)
	}
	if s := testutil.Do(t, f.r, "POST", "/auth/refresh", map[string]string{"refreshToken": a.RefreshToken}, "").Status; s != 200 {
		t.Fatalf("current session must survive, got %d", s)
	}
	if s := testutil.Do(t, f.r, "POST", "/auth/login", map[string]string{"email": email, "password": "Password!123"}, "").Status; s != 401 {
		t.Fatalf("old password still works: %d", s)
	}
	if s := testutil.Do(t, f.r, "POST", "/auth/login", map[string]string{"email": email, "password": "NewPassword!456"}, "").Status; s != 200 {
		t.Fatalf("new password rejected: %d", s)
	}
}

func TestDeleteAccount(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	a, email := f.register(t, "Password!123")
	otherID, _ := testutil.User(t, f.pool, "keep", "1992-02-02")

	key := "photos/" + a.User.ID + "/p1.jpg"
	if _, err := f.d.Storage.Put(ctx, key, bytes.NewReader([]byte("jpegdata"))); err != nil {
		t.Fatal(err)
	}
	otherKey := "photos/" + otherID + "/keep.jpg"
	if _, err := f.d.Storage.Put(ctx, otherKey, bytes.NewReader([]byte("keepdata"))); err != nil {
		t.Fatal(err)
	}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO profiles (user_id, first_name, gender) VALUES ($1,'Ann','woman')`, []any{a.User.ID}},
		{`INSERT INTO profiles (user_id, first_name, gender) VALUES ($1,'Bob','man')`, []any{otherID}},
		{`INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,0,1,1,1)`, []any{a.User.ID, key}},
		{`INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,0,1,1,1)`, []any{otherID, otherKey}},
		{`INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1,$2,'like')`, []any{a.User.ID, otherID}},
		{`INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1,$2,'like')`, []any{otherID, a.User.ID}},
		{`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1,$2)`, []any{a.User.ID, otherID}},
		{`INSERT INTO reports (reporter_id, reported_id, reason) VALUES ($1,$2,'spam')`, []any{otherID, a.User.ID}},
	} {
		if _, err := f.pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}

	del := func(pw string) int {
		return testutil.Do(t, f.r, "DELETE", "/me", map[string]string{"password": pw}, a.AccessToken).Status
	}
	if s := del("wrong-password"); s != 403 {
		t.Fatalf("wrong password want 403 got %d", s)
	}
	if s := del("Password!123"); s != 204 {
		t.Fatalf("delete want 204 got %d", s)
	}

	count := func(sql string, args ...any) int {
		var n int
		if err := f.pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	for name, n := range map[string]int{
		"users":    count(`SELECT count(*) FROM users WHERE id=$1`, a.User.ID),
		"profiles": count(`SELECT count(*) FROM profiles WHERE user_id=$1`, a.User.ID),
		"photos":   count(`SELECT count(*) FROM photos WHERE user_id=$1`, a.User.ID),
		"swipes":   count(`SELECT count(*) FROM swipes WHERE swiper_id=$1 OR target_id=$1`, a.User.ID),
		"blocks":   count(`SELECT count(*) FROM blocks WHERE blocker_id=$1 OR blocked_id=$1`, a.User.ID),
		"reports":  count(`SELECT count(*) FROM reports WHERE reporter_id=$1 OR reported_id=$1`, a.User.ID),
		"sessions": count(`SELECT count(*) FROM sessions WHERE user_id=$1`, a.User.ID),
	} {
		if n != 0 {
			t.Errorf("%s rows remain: %d", name, n)
		}
	}
	if _, err := f.d.Storage.Open(ctx, key); err == nil {
		t.Error("photo file still stored")
	}
	if rc, err := f.d.Storage.Open(ctx, otherKey); err != nil {
		t.Errorf("other user's file lost: %v", err)
	} else {
		rc.Close()
	}
	if count(`SELECT count(*) FROM users WHERE id=$1`, otherID) != 1 ||
		count(`SELECT count(*) FROM profiles WHERE user_id=$1`, otherID) != 1 ||
		count(`SELECT count(*) FROM photos WHERE user_id=$1`, otherID) != 1 {
		t.Error("other user's data damaged")
	}
	if s := testutil.Do(t, f.r, "POST", "/auth/login", map[string]string{"email": email, "password": "Password!123"}, "").Status; s != http.StatusUnauthorized {
		t.Errorf("deleted user can login: %d", s)
	}
}
