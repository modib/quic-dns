package main

import (
	"context"
	"crypto/rsa"
	"io/ioutil"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/vanderbr/logclient/zaprequest"
)

var (
	verifyKey *rsa.PublicKey
	signKey   *rsa.PrivateKey
)

func init() {
	signBytes, err := ioutil.ReadFile("/conf/jwt/jwtRS256.key")
	if err != nil {
		panic(err)
	}

	signKey, err = jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		panic(err)
	}

	verifyBytes, err := ioutil.ReadFile("/conf/jwt/jwtRS256.key.pub")
	if err != nil {
		panic(err)
	}

	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	if err != nil {
		panic(err)
	}
}

func MakeClaimsFromExisting(jwtIn jwt.MapClaims) (jwtOut jwt.MapClaims) {
	jwtOut = jwt.MapClaims{}

	for k1, v1 := range jwtIn {
		jwtOut[k1] = v1
	}
	now := time.Now().Unix()
	jwtOut["iat"] = now
	jwtOut["exp"] = now + 2*3600

	// arrnotvalid := []string{
	// 	// 	JWTFLAG_RESETPASSNEXTLOGIN,
	// 	// 	JWTFLAG_SHOWTERMS,
	// 	// 	JWTFLAG_TFASMSSETUP,
	// 	// 	JWTFLAG_TFASMSLOGIN,
	// 	// 	JWTFLAG_TFAGOOGLESETUP,
	// 	// 	JWTFLAG_TFAGOOGLELOGIN,
	// }
	// notvalid := false

	// for _, flag := range arrnotvalid {
	// 	if _, ok := jwtIn[flag]; ok {
	// 		notvalid = true
	// 	}
	// }

	// if notvalid {
	// 	// not valid
	// } else {
	// 	// valid
	// 	if role, ok := jwtIn[JWTPROP_FUTUREROLE]; ok {
	// 		jwtOut[JWTPROP_ROLE] = role
	// 	}
	// }
	return jwtOut
} // END func MakeClaimsFromExisting(jwtIn jwt.MapClaims) (jwtOut jwt.MapClaims)
func MakeClaims(now int64, id string) jwt.MapClaims {
	t := jwt.MapClaims{
		// "tenant":     fmt.Sprint(tenantId),
		// "aud":        "postgraphql",
		// "iss":        "postgraphql",
		"iat": now,
		"exp": now + 2*3600,
		"id":  id,
	}
	return t
} // END func MakeClaims(now int64, id int64, tenantId int) jwt.MapClaims

func MakeClaims3(id string, iat int64, exp int64) jwt.MapClaims {
	t := jwt.MapClaims{
		// "tenant":     fmt.Sprint(tenantId),
		// "aud":        "postgraphql",
		// "iss":        "postgraphql",
		"iat": iat,
		"exp": exp,
		"id":  id,
	}
	return t
} // END func MakeClaims3(now int64, id int64, tenantId int) jwt.MapClaims

func SignClaims(ctx context.Context, t jwt.MapClaims) (string, error) {
	logger := zaprequest.Logger(ctx)
	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, t)
	// // tt, err := token.SignedString(hs256Secret)
	// tt, err := token.SignedString([]byte(JWT_SECRET))
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, t)
	// tt, err := token.SignedString(hs256Secret)
	tt, err := token.SignedString(signKey)
	if err != nil {
		return tt, err
	}
	_, err = jwt.Parse(tt, func(token *jwt.Token) (interface{}, error) {
		return verifyKey, nil
	})
	if err != nil {
		logger.Errorw(
			"Error varifying signed JWT",
			"error", err.Error(),
		)
	}
	return tt, nil
} // END func SignClaims(t jwt.MapClaims) (string, error)
