package main

import (
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/vanderbr/logclient/util"
)

type AuthHandler struct {
	next http.Handler
}

func InitAuthHandler(next http.Handler) *AuthHandler {
	ah := &AuthHandler{
		next: next,
	}
	return ah
}

func (ah *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	requestID := uuid.New().String()

	var err error

	w.Header().Set("Access-Control-Allow-Origin", "*")
	// x-auth would be used for that service 99%
	w.Header().Set("Access-Control-Allow-Headers", "content-type,Authorization")
	w.Header().Set("Content-Type", "text/plain")

	if r.Method == "OPTIONS" {
		return
	}

	authorize := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authorize, "Bearer ")
	if authorize == token {
		// logger.Errorw(
		// 	"h_profile_profit",
		// 	"error", "Wrong authorize header format",
		// )
		util.HandleErrorUUID(
			w,
			500,
			"Error",
			requestID,
		)
		return
	}

	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// logger.Infow(
		// 	"AUTH TOKEN",
		// 	"JWT.Header", util.NiceJsonString(t.Header),
		// 	"JWT", util.NiceJsonString(t),
		// )
		// kid := pyraconv.ToString(t.Header["kid"])
		// key1, err := keyCache.Get(ctx, kid)
		// return key1, err
		return verifyKey, nil
	})
	if err != nil {
		// logger.Errorw(
		// 	"h_get_all_tokens_from_profit failed to parse jwt",
		// 	"error", err.Error(),
		// 	"Token", token,
		// )
		util.HandleErrorUUID(
			w,
			500,
			"Error",
			requestID,
		)
		return
	}

	mclaims := t.Claims.(jwt.MapClaims)

	if mclaims["type"] != "dns_token" {
		util.HandleErrorUUID(
			w,
			500,
			"Error",
			requestID,
		)
		return
	}

	ah.next.ServeHTTP(w, r)

} // END func (ah *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
