package user

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/variable"
	"github.com/jom-io/gorig/mid/tokenx"
	"github.com/jom-io/gorig/utils/errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

func Login(ctx *gin.Context, hashPwd string) (sign *string, err *errors.Error) {
	if variable.OMKey == "" {
		return nil, errors.Verify("Connection rejected")
	}

	now := time.Now().Unix() / 10
	//logger.Info(ctx, fmt.Sprintf("Login now:%d", now))
	localPwd := fmt.Sprintf("%d%s", now, variable.OMKey)
	if e := bcrypt.CompareHashAndPassword([]byte(hashPwd), []byte(localPwd)); e != nil {
		return nil, errors.Verify("Password not match")
	}

	IP := fmt.Sprintf("%s-%s", "OM", ctx.ClientIP())
	tokens, e := tokenx.Get(tokenx.Jwt, tokenx.Memory).Manager.GenerateAndRecord(IP, nil, time.Now().Unix()+3600)
	if e != nil {
		return nil, e
	}
	return &tokens, nil
}

func IsOM(userID string) bool {
	return strings.HasPrefix(userID, "OM")
}
