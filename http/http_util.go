package http

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"text/template"
)

// Resp JSON Http响应
type Resp struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Msg     string      `json:"msg"`
}

var (
	errNoparam = fmt.Errorf("missing param")
)

// GetParameter 取得由name指定的参数值
func GetParameter(r url.Values, name string) string {
	return strings.TrimSpace(r.Get(name))
}

func getIntParameter(r url.Values, name string, bitSize int) (val int64, err error) {
	value := GetParameter(r, name)
	if value == "" {
		return 0, errNoparam
	}
	val, err = strconv.ParseInt(value, 10, bitSize)
	return
}

// GetInt64Parameter 取得由name指定的64位整数参数值
func GetInt64Parameter(r url.Values, name string) (val int64, err error) {
	val, err = getIntParameter(r, name, 64)
	return
}

// GetInt32Parameter 取得由name指定的32位整数参数值
func GetInt32Parameter(r url.Values, name string) (val int32, err error) {
	val64, err := getIntParameter(r, name, 32)
	if err == nil {
		return int32(val64), nil
	}
	return 0, err
}

// RenderTemplate 渲染模板
func RenderTemplate(w http.ResponseWriter, templateDir, tmpl string, data interface{}) {
	templatePath := path.Join(templateDir, tmpl+".html")
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("Parse template err:%s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := t.Execute(w, data); err != nil {
		log.Printf("Execute template err:%s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// RenderJSON 渲染JSON
func RenderJSON(w http.ResponseWriter, jsonData interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(jsonData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// RenderText 渲染Text
func RenderText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(text))
}

// GetURL 请求URL
func GetURL(client *http.Client, url string, params url.Values) (string, error) {
	return GetURLWithCookie(client, url, params, nil)
}

// GetURLWithCookie 请求URL
func GetURLWithCookie(client *http.Client, url string, params url.Values, cookies map[string]string) (string, error) {
	var req *http.Request
	var err error
	if params != nil {
		req, err = http.NewRequest("GET", url+"?"+params.Encode(), nil)
	} else {
		req, err = http.NewRequest("GET", url, nil)
	}
	if err != nil {
		return "", err
	}
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Status:%d,msg:%s", resp.StatusCode, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// PostURL 请求URL
func PostURL(client *http.Client, url string, params url.Values, contentType string, requestBody io.Reader) ([]byte, http.Header, error) {
	return PostURLWithCookie(client, url, params, contentType, requestBody, nil)
}

// PostURLWithCookie 请求URL
func PostURLWithCookie(client *http.Client, url string, params url.Values, contentType string, requestBody io.Reader, cookies map[string]string) ([]byte, http.Header, error) {
	if requestBody == nil {
		requestBody = strings.NewReader(params.Encode())
	} else {
		url = url + "?" + params.Encode()
	}

	req, err := http.NewRequest("POST", url, requestBody)
	if err != nil {
		return nil, nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("Status:%d,msg:%s", resp.StatusCode, resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return body, resp.Header, nil
}
