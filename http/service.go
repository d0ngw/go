package http

import (
	"github.com/d0ngw/go/common/perm"
)

//AuthService 验证服务
type AuthService interface {
	//AuthToken 使用token认证
	AuthToken(token string) (principal perm.Principal, err error)
}
