package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	c "github.com/d0ngw/go/common"
)

// RequestError is the http request response error
type RequestError struct {
	Status int
	Err    error
}

func (p *RequestError) Error() string {
	return fmt.Sprintf("status:%d,data:%s", p.Status, p.Err)
}

// CheckRequestError if err is RequestError,then return response status code
func CheckRequestError(err error) (status int, ok bool) {
	if err == nil {
		return
	}
	if e, ok := err.(*RequestError); ok {
		return e.Status, true
	}
	return
}

// RedirectError redirect error
type RedirectError struct {
	RedirectURL string
}

func (p *RedirectError) Error() string {
	return fmt.Sprintf("redirect:%s", p.RedirectURL)
}

// CheckRedirectError if err is RedirectError ,then return redirect error
func CheckRedirectError(err error) (url string, ok bool) {
	if err == nil {
		return
	}
	if e, ok := err.(*RedirectError); ok {
		return e.RedirectURL, true
	}
	return
}

// NewRedirectError redirect error
func NewRedirectError(req *http.Request, via []*http.Request) error {
	return &RedirectError{
		RedirectURL: req.URL.String(),
	}
}

// Resp JSON Http响应
type Resp struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Msg     string      `json:"msg"`
}

// NewSuccResp 构建成功的响应
func NewSuccResp(data interface{}) *Resp {
	return &Resp{
		Success: true,
		Data:    data,
	}
}

// NewFailResp 构建失败的响应
func NewFailResp(msg string) *Resp {
	return &Resp{
		Success: false,
		Msg:     msg,
	}
}

// ResponseHandler 响应处理
type ResponseHandler struct {
	Success bool
	Data    interface{}
	Msg     string
	w       http.ResponseWriter
}

// SuccessWithData 设置成功及数据
func (p *ResponseHandler) SuccessWithData(data interface{}) {
	p.Success = true
	p.Data = data
}

// FailWithMsg 设置失败及出错的消息
func (p *ResponseHandler) FailWithMsg(msg string) {
	p.Success = false
	p.Msg = msg
}

func (p *ResponseHandler) Run() {
	if !p.Success {
		RenderJSON(p.w, &Resp{Success: false, Msg: p.Msg})
	} else {
		RenderJSON(p.w, &Resp{Success: true, Data: p.Data})
	}
}

// NewResponseHandler 构建响应处理
func NewResponseHandler(w http.ResponseWriter) *ResponseHandler {
	return &ResponseHandler{
		w: w,
	}
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

func getFloatParameter(r url.Values, name string, bitSize int) (val float64, err error) {
	value := GetParameter(r, name)
	if value == "" {
		return 0, errNoparam
	}
	val, err = strconv.ParseFloat(value, bitSize)
	return
}

// GetFloat64Parameter 取得由name指定的64位浮点数参数值
func GetFloat64Parameter(r url.Values, name string) (val float64, err error) {
	val, err = getFloatParameter(r, name, 64)
	return
}

// GetFloat32Parameter 取得由name指定的32位浮点数参数值
func GetFloat32Parameter(r url.Values, name string) (val float32, err error) {
	val64, err := getFloatParameter(r, name, 32)
	if err == nil {
		return float32(val64), nil
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
	_, body, err := GetURLRaw(client, url, params, nil, cookies)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// GetURLRawToWriter 请求URL
func GetURLRawToWriter(client *http.Client, url string, params url.Values, reqHeader http.Header, cookies map[string]string, writer io.Writer) (header http.Header, err error) {
	var req *http.Request
	if params != nil {
		req, err = http.NewRequest("GET", url+"?"+params.Encode(), nil)
	} else {
		req, err = http.NewRequest("GET", url, nil)
	}
	if err != nil {
		return nil, err
	}
	if reqHeader != nil {
		req.Header = reqHeader
	}
	for k, v := range cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	resp, err := client.Do(req)
	if err != nil {
		e := &RequestError{Err: err}
		if resp != nil {
			e.Status = resp.StatusCode
		}
		return nil, e
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		return nil, &RequestError{Status: resp.StatusCode}
	}
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch contentEncoding {
	case "gzip":
		fallthrough
	case "deflate":
		var (
			reader io.ReadCloser
			err    error
		)
		if contentEncoding == "gzip" {
			reader, err = gzip.NewReader(resp.Body)
		} else if contentEncoding == "deflate" {
			reader = flate.NewReader(resp.Body)
		}
		if err != nil {
			return nil, err
		}
		if reader == nil {
			return nil, fmt.Errorf("no uncompress reader")
		}
		defer reader.Close()
		_, err = io.Copy(writer, reader)
		if err != nil {
			return nil, err
		}
	default:
		_, err = io.Copy(writer, resp.Body)
		if err != nil {
			return nil, err
		}
	}
	return resp.Header, nil
}

// GetURLRaw 请求URL
func GetURLRaw(client *http.Client, url string, params url.Values, reqHeader http.Header, cookies map[string]string) (header http.Header, body []byte, err error) {
	var writer = &bytes.Buffer{}
	header, err = GetURLRawToWriter(client, url, params, reqHeader, cookies, writer)
	if err != nil {
		return header, nil, err
	}
	return header, writer.Bytes(), nil
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
		return nil, nil, &RequestError{Status: resp.StatusCode}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return body, resp.Header, nil
}

// ParseParams 从r中解析参数,并填充到dest中,params应该是struct指针
func ParseParams(r url.Values, dest interface{}) error {
	if r == nil || dest == nil {
		return fmt.Errorf("invalid args")
	}

	val, ind, typ := c.ExtractRefTuple(dest)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("Expect ptr ,but it's %s", val.Kind())
	}
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("Expect struct,but it's %s", typ.Kind())
	}

	for i := 0; i < ind.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag
		paramName := tag.Get("pname")
		if paramName == "" {
			paramName = ToUnderlineName(field.Name)
		}

		if paramName == "_" {
			continue
		}

		fieldVal := ind.Field(i)
		switch field.Type.Kind() {
		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			if val64, err := getIntParameter(r, paramName, 64); err == nil {
				fieldVal.SetInt(val64)
			}
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			if val64, err := getFloatParameter(r, paramName, 64); err == nil {
				fieldVal.SetFloat(val64)
			}
		case reflect.String:
			fieldVal.SetString(GetParameter(r, paramName))
		case reflect.Bool:
			v := strings.ToLower(GetParameter(r, paramName))
			fieldVal.SetBool(v == "1" || v == "y" || v == "true")
		case reflect.Slice:
			paramSep := tag.Get("psep")
			var vals []string
			if paramSep == "" {
				if ps, ok := r[paramName]; ok {
					vals = c.TrimOmitEmpty(ps)
				}
			} else {
				vals = c.SplitTrimOmitEmpty(GetParameter(r, paramName), paramSep)
			}

			strSlice := c.StringSlice(vals)
			var elem = field.Type.Elem()
			switch elem.Kind() {
			case reflect.String:
				fieldVal.Set(reflect.ValueOf(vals))
			case reflect.Int32:
				if intSlice, err := strSlice.ToInt32(); err == nil {
					fieldVal.Set(reflect.ValueOf(intSlice))
				}
			case reflect.Int64:
				if intSlice, err := strSlice.ToInt64(); err == nil {
					fieldVal.Set(reflect.ValueOf(intSlice))
				}
			case reflect.Float32:
				if intSlice, err := strSlice.ToFloat32(); err == nil {
					fieldVal.Set(reflect.ValueOf(intSlice))
				}
			case reflect.Float64:
				if intSlice, err := strSlice.ToFloat64(); err == nil {
					fieldVal.Set(reflect.ValueOf(intSlice))
				}
			default:
				return fmt.Errorf("Unsupported type %s", elem.Kind())
			}
		case reflect.Struct:
			if fieldVal.IsValid() && fieldVal.CanInterface() && fieldVal.CanAddr() {
				ParseParams(r, fieldVal.Addr().Interface())
			} else {
				return fmt.Errorf("Unsupported struct type %s", fieldVal.Type())
			}
		default:
			return fmt.Errorf("Unsupported field type %s", field.Type.Kind())
		}
	}
	return nil
}
