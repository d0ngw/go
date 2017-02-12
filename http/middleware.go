package http

import (
	"net/http"
	"strings"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/common/perm"
)

// MiddlewareFunc middleware函数
type MiddlewareFunc func(http.ResponseWriter, *http.Request)

// Middleware 接口
type Middleware interface {
	// Handle处理
	Handle(next MiddlewareFunc) MiddlewareFunc
}

// RequestMehotdMiddleware http请求方法
type RequestMehotdMiddleware struct {
	//允许的请求方法
	AllowsMethods map[string]struct{}
}

// Handle 校验Http请求的方法
func (p *RequestMehotdMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := p.AllowsMethods[r.Method]; !ok {
			http.Error(w, "Bad Request", http.StatusMethodNotAllowed)
			return
		}
		next(w, r)
	}
}

// NewRequestMetodMiddleware 用methods构建middleware
func NewRequestMetodMiddleware(methods ...string) *RequestMehotdMiddleware {
	m := &RequestMehotdMiddleware{
		AllowsMethods: map[string]struct{}{},
	}
	for _, method := range methods {
		m.AllowsMethods[strings.ToUpper(method)] = struct{}{}
	}
	return m
}

// PermBindMiddleware 用于给controller中的handler方法注入需要的权限
type PermBindMiddleware struct {
	ReqPerm []*perm.Perm
}

// Handle 绑定权限到ctx
func (p *PermBindMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if p.ReqPerm != nil && len(p.ReqPerm) > 0 {
			ctx, _ := perm.ReqPerm(r.Context(), p.ReqPerm)
			r = r.WithContext(ctx)
		}
		next(w, r)
	}
}

// NewPermBindMiddleware 用perms构建middleware
func NewPermBindMiddleware(perms ...*perm.Perm) *PermBindMiddleware {
	m := &PermBindMiddleware{
		ReqPerm: []*perm.Perm{},
	}
	for _, p := range perms {
		if p != nil {
			m.ReqPerm = append(m.ReqPerm, p)
		}
	}
	return m
}

// TokenMiddleware 用于从cookie中解析token,从中取得请求的principal
type TokenMiddleware struct {
	TokenName   string      //Token的名称
	AuthService AuthService `inject:"_"` //认证服务
}

// Handle 解析token
func (p *TokenMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenCookie, err := r.Cookie(p.TokenName)
		if err == nil {
			//检查是否已经有principal了
			principal, _ := perm.GetPrincipal(r.Context())
			if principal == nil {
				authUser, err := p.AuthService.AuthToken(tokenCookie.Value)
				if err != nil {
					c.Errorf("auth by token %s fail,err:%s", tokenCookie.Value, err)
				} else {
					ctx, _ := perm.BindPrincipal(r.Context(), authUser)
					r = r.WithContext(ctx)
				}
			}
		}
		next(w, r)
	}
}

// CheckPermMiddleware 用于检查用户的权限
type CheckPermMiddleware struct {
	AuthService AuthService `inject:"_"` //认证服务
}

// Handle 检查权限
func (p *CheckPermMiddleware) Handle(next MiddlewareFunc) MiddlewareFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		principal, _ := perm.GetPrincipal(r.Context())
		if !perm.HasPermWithPrinciapl(ctx, principal) {
			RenderJSON(w, &Resp{Msg: "No permission"})
		} else {
			next(w, r)
		}
	}
}
