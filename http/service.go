package http

import (
	"github.com/d0ngw/go/common/perm"
)

//AuthService 验证服务
type AuthService interface {
	//AuthToken 使用token认证
	AuthToken(token string) (principal perm.Principal, err error)
}

// PermServcie  权限服务
type PermServcie interface {
	//HasPerm 坚持prinpical是否有需要的权限
	HasPerm(principal perm.Principal, perms []*perm.Perm) (hasPerm bool, err error)
}
