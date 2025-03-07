package tokenx

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/jom-io/gorig/global/errc"
	"github.com/jom-io/gorig/utils/errors"
	"time"
)

type jwtGenerator struct {
	SigningKey []byte
}

func (j *jwtGenerator) Generate(userId string, userInfo map[string]interface{}, expireAt int64) (tokens string, err *errors.Error) {
	claims := CustomClaims{
		UserId:   userId,
		UserInfo: userInfo,
		StandardClaims: jwt.StandardClaims{
			NotBefore: time.Now().Unix() - 10,
			ExpiresAt: time.Now().Unix() + expireAt,
		},
	}
	tokenPartA := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if signedString, sinErr := tokenPartA.SignedString(j.SigningKey); sinErr == nil {
		return signedString, nil
	} else {
		return "", errors.Sys("jwt sign string error", sinErr)
	}
}

func (j *jwtGenerator) Parse(token string) (*CustomClaims, *errors.Error) {
	if customClaims, err := j.ParseToken(token); err == nil {
		return customClaims, nil
	} else {
		return &CustomClaims{}, err
	}
}

func (j *jwtGenerator) ParseToken(tokenString string) (*CustomClaims, *errors.Error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if token == nil {
		return nil, errors.Verify(errc.ErrorsTokenInvalid)
	}
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.Verify(errc.ErrorsTokenMalFormed)
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.Verify(errc.ErrorsTokenNotActiveYet)
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				token.Valid = true
				goto labelHere
			} else {
				return nil, errors.Verify(errc.ErrorsTokenInvalid)
			}
		}
	}
labelHere:
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, errors.Verify(errc.ErrorsTokenInvalid)
	}
}
