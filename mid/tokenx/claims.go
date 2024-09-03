package tokenx

import "github.com/dgrijalva/jwt-go"

type CustomClaims struct {
	UserId   string
	UserInfo map[string]interface{}
	jwt.StandardClaims
}
