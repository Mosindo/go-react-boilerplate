package photos_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/features/photos"
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
	r := testutil.Router(func(r gin.IRouter) { photos.RegisterRoutes(r, pool, d) })
	return fixture{pool: pool, d: d, r: r}
}

func jpegBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = byte(i * 7)
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

func (f fixture) upload(t *testing.T, token, field, filename string, data []byte) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/me/photos", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.r.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

type photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

func (f fixture) mustUpload(t *testing.T, token string) photo {
	t.Helper()
	s, b := f.upload(t, token, "file", "a.jpg", jpegBytes(120, 80))
	if s != 201 {
		t.Fatalf("upload %d %s", s, b)
	}
	var p photo
	if err := jsonUnmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func (f fixture) get(path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	f.r.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestUploadAndServe(t *testing.T) {
	f := setup(t)
	id, tok := testutil.User(t, f.pool, "up", "1995-01-01")

	if s, _ := f.upload(t, "", "file", "a.jpg", jpegBytes(10, 10)); s != 401 {
		t.Fatalf("unauth want 401 got %d", s)
	}
	// Large image is resized, stored as JPEG, and the row + object exist.
	s, b := f.upload(t, tok, "file", "../../evil.php", jpegBytes(2400, 1200))
	if s != 201 {
		t.Fatalf("upload %d %s", s, b)
	}
	var p photo
	_ = jsonUnmarshal(b, &p)
	if p.Position != 0 || !strings.HasPrefix(p.URL, "/photos/"+p.ID+"/file?exp=") {
		t.Fatalf("unexpected photo %+v", p)
	}
	var key string
	var w, h int
	if err := f.pool.QueryRow(context.Background(), `SELECT storage_key, width, height FROM photos WHERE id=$1`, p.ID).Scan(&key, &w, &h); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "photos/"+id+"/") || !strings.HasSuffix(key, ".jpg") || strings.Contains(key, "evil") || w != 1080 || h != 540 {
		t.Fatalf("key=%q %dx%d", key, w, h)
	}

	// Signed URL: valid.
	rec := f.get(p.URL)
	if rec.Code != 200 {
		t.Fatalf("file %d %s", rec.Code, rec.Body.String())
	}
	hd := rec.Header()
	if hd.Get("Content-Type") != "image/jpeg" || hd.Get("Cache-Control") != "private, max-age=3600" || hd.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers %v", hd)
	}
	img, err := jpeg.Decode(rec.Body)
	if err != nil || img.Bounds().Dx() != 1080 {
		t.Fatalf("served body not the processed jpeg: %v", err)
	}

	// Tampered signature / id / exp, missing params, expired, unknown id: all 404.
	u, _ := url.Parse(p.URL)
	q := u.Query()
	exp, sig := q.Get("exp"), q.Get("sig")
	other := f.mustUpload(t, tok)
	expired, expiredSig := "", ""
	{
		// forge an expired-but-correctly-signed link by signing with a signer whose clock is in the past
		// is impossible from outside; instead reuse a valid sig with a smaller exp (signature no longer matches).
		expired = fmt.Sprintf("%d", time.Now().Add(-time.Hour).Unix())
		expiredSig = sig
	}
	bad := []string{
		fmt.Sprintf("/photos/%s/file?exp=%s&sig=%s", p.ID, exp, strings.Repeat("0", 64)),
		fmt.Sprintf("/photos/%s/file?exp=%s&sig=%s", p.ID, exp, sig[:len(sig)-1]+flip(sig[len(sig)-1])),
		fmt.Sprintf("/photos/%s/file?exp=%s&sig=%s", other.ID, exp, sig), // signature of another photo
		fmt.Sprintf("/photos/%s/file?exp=%s&sig=%s", p.ID, expired, expiredSig),
		fmt.Sprintf("/photos/%s/file?exp=%d&sig=%s", p.ID, time.Now().Unix()+999999, sig),
		fmt.Sprintf("/photos/%s/file?exp=abc&sig=%s", p.ID, sig),
		fmt.Sprintf("/photos/%s/file", p.ID),
		"/photos/not-a-uuid/file?exp=1&sig=x",
	}
	for _, path := range bad {
		if rec := f.get(path); rec.Code != 404 {
			t.Errorf("%s want 404 got %d", path, rec.Code)
		}
	}
	// Correctly signed, but the photo does not exist -> 404.
	ghost := "00000000-0000-4000-8000-000000000000"
	if rec := f.get(f.d.Signer.PhotoURL(ghost)); rec.Code != 404 {
		t.Errorf("ghost photo want 404 got %d", rec.Code)
	}
}

func TestUploadRejects(t *testing.T) {
	f := setup(t)
	_, tok := testutil.User(t, f.pool, "rej", "1995-01-01")

	if s, _ := f.upload(t, tok, "file", "x.jpg", []byte("<?php echo 1; ?>")); s != 400 {
		t.Errorf("non-image want 400 got %d", s)
	}
	// A real JPEG announced under a misleading name still works (sniffing, not extension)...
	if s, b := f.upload(t, tok, "file", "photo.txt", jpegBytes(50, 50)); s != 201 {
		t.Errorf("sniffed jpeg want 201 got %d %s", s, b)
	}
	// ...and a script named .jpg does not.
	if s, _ := f.upload(t, tok, "file", "photo.jpg", []byte("GIF89a....")); s != 400 {
		t.Errorf("gif want 400 got %d", s)
	}
	if s, _ := f.upload(t, tok, "wrongfield", "a.jpg", jpegBytes(50, 50)); s != 400 {
		t.Errorf("wrong field want 400 got %d", s)
	}
	// Oversize: over 5 MB is 413 and nothing is stored.
	big := append(jpegBytes(50, 50), bytes.Repeat([]byte{0}, photos.MaxUploadBytes)...)
	if s, _ := f.upload(t, tok, "file", "big.jpg", big); s != 413 {
		t.Errorf("oversize want 413 got %d", s)
	}
	// Non-multipart body.
	res := testutil.Do(t, f.r, "POST", "/me/photos", map[string]string{"a": "b"}, tok)
	if res.Status != 400 {
		t.Errorf("json body want 400 got %d", res.Status)
	}
	var n int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos p JOIN users u ON u.id=p.user_id WHERE u.email LIKE 'rej\_%'`).Scan(&n)
	if n != 1 {
		t.Errorf("only the sniffed jpeg should be stored, got %d rows", n)
	}
}

func TestPhotoLimit(t *testing.T) {
	f := setup(t)
	id, tok := testutil.User(t, f.pool, "lim", "1995-01-01")
	for i := 0; i < photos.MaxPhotosPerUser; i++ {
		p := f.mustUpload(t, tok)
		if p.Position != i {
			t.Fatalf("position %d want %d", p.Position, i)
		}
	}
	s, b := f.upload(t, tok, "file", "a.jpg", jpegBytes(30, 30))
	if s != 422 {
		t.Fatalf("7th photo want 422 got %d %s", s, b)
	}
	if n := countFiles(t, f, id); n != 6 {
		t.Fatalf("stored objects = %d, want 6 (no orphan for the rejected upload)", n)
	}
}

func countFiles(t *testing.T, f fixture, userID string) int {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id=$1`, userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		rc, err := f.d.Storage.Open(context.Background(), k)
		if err == nil {
			_, _ = io.Copy(io.Discard, rc)
			rc.Close()
			n++
		}
	}
	return n
}

func TestConcurrentUploadsRespectLimit(t *testing.T) {
	f := setup(t)
	id, tok := testutil.User(t, f.pool, "conc", "1995-01-01")
	for i := 0; i < 4; i++ {
		f.mustUpload(t, tok)
	}
	data := jpegBytes(40, 40)
	codes := make(chan int, 6)
	for i := 0; i < 6; i++ {
		go func() {
			s, _ := f.upload(t, tok, "file", "a.jpg", data)
			codes <- s
		}()
	}
	created := 0
	for i := 0; i < 6; i++ {
		if <-codes == 201 {
			created++
		}
	}
	var n int
	_ = f.pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE user_id=$1`, id).Scan(&n)
	if created != 2 || n != 6 {
		t.Fatalf("created=%d total=%d, want 2 and 6", created, n)
	}
	if files := countFiles(t, f, id); files != 6 {
		t.Fatalf("files=%d", files)
	}
}

func TestReorderAndDelete(t *testing.T) {
	f := setup(t)
	_, tok := testutil.User(t, f.pool, "ord", "1995-01-01")
	_, otherTok := testutil.User(t, f.pool, "ord2", "1995-01-01")
	a, b, c := f.mustUpload(t, tok), f.mustUpload(t, tok), f.mustUpload(t, tok)
	foreign := f.mustUpload(t, otherTok)

	reorder := func(ids ...string) (int, []byte) {
		res := testutil.Do(t, f.r, "PUT", "/me/photos/order", map[string]any{"photoIds": ids}, tok)
		return res.Status, res.Body
	}
	for name, ids := range map[string][]string{
		"missing one":   {a.ID, b.ID},
		"duplicate":     {a.ID, a.ID, b.ID},
		"extra":         {a.ID, b.ID, c.ID, foreign.ID},
		"foreign photo": {a.ID, b.ID, foreign.ID},
		"empty":         {},
	} {
		if s, body := reorder(ids...); s != 422 {
			t.Errorf("%s: want 422 got %d %s", name, s, body)
		}
	}
	if s, _ := reorder("nope", a.ID, b.ID); s != 400 {
		t.Errorf("malformed id want 400 got %d", s)
	}

	s, body := reorder(c.ID, a.ID, b.ID)
	if s != 200 {
		t.Fatalf("reorder %d %s", s, body)
	}
	var out struct {
		Photos []photo `json:"photos"`
	}
	_ = jsonUnmarshal(body, &out)
	if len(out.Photos) != 3 || out.Photos[0].ID != c.ID || out.Photos[1].ID != a.ID || out.Photos[2].ID != b.ID ||
		out.Photos[0].Position != 0 || out.Photos[2].Position != 2 {
		t.Fatalf("order: %s", body)
	}

	// Delete: foreign -> 404 and untouched; own -> 204, repacked, file gone.
	if s := testutil.Do(t, f.r, "DELETE", "/me/photos/"+foreign.ID, nil, tok).Status; s != 404 {
		t.Fatalf("delete foreign want 404 got %d", s)
	}
	if s := testutil.Do(t, f.r, "DELETE", "/me/photos/not-a-uuid", nil, tok).Status; s != 404 {
		t.Fatalf("delete malformed want 404 got %d", s)
	}
	var key string
	_ = f.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id=$1`, a.ID).Scan(&key)
	if s := testutil.Do(t, f.r, "DELETE", "/me/photos/"+a.ID, nil, tok).Status; s != 204 {
		t.Fatalf("delete want 204 got %d", s)
	}
	if _, err := f.d.Storage.Open(context.Background(), key); err == nil {
		t.Fatal("file still stored")
	}
	if rec := f.get(a.URL); rec.Code != 404 {
		t.Fatalf("deleted photo still served: %d", rec.Code)
	}
	rows, _ := f.pool.Query(context.Background(), `SELECT id, position FROM photos WHERE user_id=(SELECT user_id FROM photos WHERE id=$1) ORDER BY position`, c.ID)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var id string
		var pos int
		_ = rows.Scan(&id, &pos)
		got = append(got, fmt.Sprintf("%s@%d", id, pos))
	}
	want := []string{c.ID + "@0", b.ID + "@1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("after delete: %v want %v", got, want)
	}
	if rec := f.get(foreign.URL); rec.Code != http.StatusOK {
		t.Fatalf("other user's photo affected: %d", rec.Code)
	}
	// Slot freed: upload works again and lands at the end.
	if p := f.mustUpload(t, tok); p.Position != 2 {
		t.Fatalf("new position %d want 2", p.Position)
	}
}
