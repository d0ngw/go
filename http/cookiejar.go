package http

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"

	c "github.com/d0ngw/go/common"
)

// RetrivedCookieJar 可持久化的Cookie
type RetrivedCookieJar struct {
	jar  *cookiejar.Jar
	urls map[string]struct{}
	mu   sync.Mutex
}

// NewRetrivedCookieJar 构建PersistCookieJar
func NewRetrivedCookieJar(o *cookiejar.Options) *RetrivedCookieJar {
	jar, _ := cookiejar.New(o)
	return &RetrivedCookieJar{
		jar:  jar,
		urls: map[string]struct{}{},
	}
}

// SetCookies implements CookieJar.SetCookies
func (p *RetrivedCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	p.jar.SetCookies(u, cookies)
	cookieURL := u.String()
	p.mu.Lock()
	p.urls[cookieURL] = struct{}{}
	p.mu.Unlock()
}

// Cookies implements CookeJar.Cookies
func (p *RetrivedCookieJar) Cookies(u *url.URL) []*http.Cookie {
	return p.jar.Cookies(u)
}

// URLAndCookies 取得所有的URL和Cookie
func (p *RetrivedCookieJar) URLAndCookies() map[string][]*http.Cookie {
	all := map[string][]*http.Cookie{}
	for u := range p.urls {
		cookieURL, err := url.Parse(u)
		if err != nil {
			c.Errorf("parse %s fail,err:%s", u, err)
			continue
		}

		cookies := p.Cookies(cookieURL)
		if cookies != nil {
			all[u] = cookies
		}
	}
	return all
}

// SetURLAndCookies 设置所有的URL和Cookie
func (p *RetrivedCookieJar) SetURLAndCookies(all map[string][]*http.Cookie) error {
	if all == nil {
		return nil
	}

	for u, cookies := range all {
		if cookies == nil {
			continue
		}

		cookieURL, err := url.Parse(u)
		if err != nil {
			c.Errorf("parse %s fail,err:%s", u, err)
			continue
		}
		p.SetCookies(cookieURL, cookies)
	}

	return nil
}
