// Package httpx regroupe les helpers HTTP partages (reponses d'erreur, pagination).
package httpx

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Error ecrit une reponse d'erreur JSON uniforme: {"error": "..."}.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// Internal ecrit une erreur 500 generique (le detail est journalise, jamais expose).
func Internal(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "internal error")
}

// Limit lit le parametre `limit` avec valeur par defaut et plafond.
func Limit(c *gin.Context, def, max int) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > max {
		return 0, false
	}
	return n, true
}

// IsUUID verifie qu'une chaine est un UUID canonique (evite les erreurs SQL de cast).
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			if s[i] != '-' {
				return false
			}
		default:
			ch := s[i]
			if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
				return false
			}
		}
	}
	return true
}

// ParamUUID lit un parametre de chemin UUID; ecrit un 404 et renvoie false sinon.
func ParamUUID(c *gin.Context, name string) (string, bool) {
	v := c.Param(name)
	if !IsUUID(v) {
		Error(c, http.StatusNotFound, "not found")
		return "", false
	}
	return v, true
}

// Cursor est un curseur de pagination keyset (horodatage + id), encode en texte opaque.
type Cursor struct {
	At time.Time
	ID string
}

// EncodeCursor produit un curseur opaque.
func EncodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(at.UnixMicro(), 10) + "|" + id))
}

// DecodeCursor lit le parametre `cursor`. ok=false si le curseur est invalide;
// (nil, true) si absent.
func DecodeCursor(c *gin.Context) (cur *Cursor, ok bool) {
	raw := c.Query("cursor")
	if raw == "" {
		return nil, true
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, false
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 || !IsUUID(parts[1]) {
		return nil, false
	}
	us, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, false
	}
	return &Cursor{At: time.UnixMicro(us).UTC(), ID: parts[1]}, true
}
