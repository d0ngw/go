package http

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestCookieJar(t *testing.T) {
	jar := NewRetrivedCookieJar(nil)
	u, _ := url.Parse("https://www.google.com/tt")
	cookie1 := &http.Cookie{Name: "c", Value: "d"}
	cookie2 := &http.Cookie{Name: "e", Value: "f"}
	jar.SetCookies(u, []*http.Cookie{cookie1, cookie2})

	u2, _ := url.Parse("https://www.google.com/t2")
	jar.SetCookies(u2, []*http.Cookie{cookie1, cookie2})

	all := jar.URLAndCookies()
	for u, cookies := range all {
		fmt.Printf("u:%s", u)
		for _, c := range cookies {
			fmt.Printf(" cookie:%s", c)
		}
		fmt.Println()
	}

	jar = NewRetrivedCookieJar(nil)
	jar.SetURLAndCookies(all)
	all = jar.URLAndCookies()
	for u, cookies := range all {
		fmt.Printf("u:%s", u)
		for _, c := range cookies {
			fmt.Printf(" cookie:%s", c)
		}
		fmt.Println()
	}
}
