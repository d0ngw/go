package common

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// ExtractRefTuple returns the reflected value, its indirect value, and the indirect type.
func ExtractRefTuple(obj interface{}) (val reflect.Value, ind reflect.Value, typ reflect.Type) {
	val = reflect.ValueOf(obj)
	ind = reflect.Indirect(val)
	typ = ind.Type()
	return
}

func getFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	val := reflect.Indirect(reflect.ValueOf(structObj))
	return val.Field(fieldIndex).Type()
}

// GetFieldType returns the type of the specified struct field.
func GetFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	return getFieldType(structObj, fieldIndex)
}

// GetFirstFieldType returns the type of the first struct field.
func GetFirstFieldType(structObj interface{}) reflect.Type {
	return getFieldType(structObj, 0)
}

// Shutdownhook runs registered callbacks when the process shuts down.
type Shutdownhook struct {
	ch         chan os.Signal // Channel receiving OS signals.
	hooks      []func()       // Callbacks to run during shutdown.
	sync.Mutex                // Synchronization lock.
	stop       int32
	osStop     int32
}

// NewShutdownhook watches the supplied signals, defaulting to SIGINT and SIGTERM.
func NewShutdownhook(sig ...os.Signal) *Shutdownhook {
	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	ch := make(chan os.Signal, len(sig))
	signal.Notify(ch, sig...)
	return &Shutdownhook{ch: ch}
}

func (p *Shutdownhook) getCh() chan os.Signal {
	p.Lock()
	defer p.Unlock()
	return p.ch
}

// AddHook registers a shutdown hook.
func (p *Shutdownhook) AddHook(hookFunc func()) {
	p.Lock()
	defer p.Unlock()
	p.hooks = append(p.hooks, hookFunc)
}

// WaitShutdown waits for a shutdown signal and runs the registered hooks in order.
func (p *Shutdownhook) WaitShutdown() {
	ch := p.getCh()
	if ch == nil {
		return
	}

	if s, ok := <-ch; ok {
		Infof("Receive signal:%v,Run hooks", s)
		atomic.StoreInt32(&p.osStop, 1)
		p.Stop()
	} else {
		Warnf("Receive signal error,%v", ok)
	}

	p.Lock()
	defer p.Unlock()
	Infof("begin run hooks")
	for _, f := range p.hooks {
		f()
	}
	Infof("finished run hooks")
	if logger != nil {
		logger.Sync()
	}
}

// Stop the shutdownhook
func (p *Shutdownhook) Stop() {
	p.Lock()
	defer p.Unlock()
	ch := p.ch
	if ch != nil {
		signal.Stop(p.ch)
		close(p.ch)
	}
	p.ch = nil
	p.stop = 1
}

// IsOSStop reports whether shutdown was triggered by an OS signal.
func (p *Shutdownhook) IsOSStop() bool {
	return atomic.LoadInt32(&p.osStop) == 1
}

// RandomUUID generates a random version 4 UUID.
func RandomUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		Errorf("UUID Err:%s", err)
		return
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	uuid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

// FileBasename returns the file name without its extension.
func FileBasename(s string) string {
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}

// SplitTrimOmitEmpty splits str by sep, trims whitespace, and omits empty values.
func SplitTrimOmitEmpty(str, sep string) []string {
	return TrimOmitEmpty(strings.Split(str, sep))
}

// TrimOmitEmpty trims whitespace and omits empty values.
func TrimOmitEmpty(str []string) []string {
	var ret = make([]string, 0, len(str))
	for _, item := range str {
		item = strings.TrimSpace(item)
		if item != "" {
			ret = append(ret, item)
		}
	}
	return ret
}

// StringSlice string slice
type StringSlice []string

// ToInt converts values to an int slice.
func (p StringSlice) ToInt() ([]int, error) {
	if p == nil {
		return nil, nil
	}
	ret := make([]int, 0, len(p))
	for _, item := range p {
		if val64, err := strconv.ParseInt(item, 10, 64); err == nil {
			ret = append(ret, int(val64))
		} else {
			return nil, err
		}
	}
	return ret, nil
}

// ToNumber converts values to a numeric slice.
func (p StringSlice) ToNumber(typ interface{}) (interface{}, error) {
	sliceType := reflect.TypeOf(typ)
	intSlice := reflect.MakeSlice(sliceType, 0, len(p))
	sliceElemType := sliceType.Elem()
	k := sliceElemType.Kind()

	var toAdd = make([]reflect.Value, 0, len(p))
	for _, item := range p {
		v := reflect.New(sliceElemType)
		indV := reflect.Indirect(v)
		switch k {
		case reflect.Int:
			fallthrough
		case reflect.Int8:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			if val64, err := strconv.ParseInt(item, 10, 64); err == nil {
				indV.SetInt(val64)
				toAdd = append(toAdd, indV)
			} else {
				return nil, err
			}
		case reflect.Uint:
			fallthrough
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			if val64, err := strconv.ParseUint(item, 10, 64); err == nil {
				indV.SetUint(val64)
				toAdd = append(toAdd, indV)
			} else {
				return nil, err
			}
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			if val64, err := strconv.ParseFloat(item, 64); err == nil {
				indV.SetFloat(val64)
				toAdd = append(toAdd, indV)
			} else {
				return nil, err
			}
		}
	}
	intSlice = reflect.Append(intSlice, toAdd...)
	return intSlice.Interface(), nil
}

// ToInt8 converts values to an int8 slice.
func (p StringSlice) ToInt8() ([]int8, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]int8{})
	if err != nil {
		return nil, err
	}
	return val.([]int8), nil
}

// ToInt16 converts values to an int16 slice.
func (p StringSlice) ToInt16() ([]int16, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]int16{})
	if err != nil {
		return nil, err
	}
	return val.([]int16), nil
}

// ToInt32 converts values to an int32 slice.
func (p StringSlice) ToInt32() ([]int32, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]int32{})
	if err != nil {
		return nil, err
	}
	return val.([]int32), nil
}

// ToInt64 converts values to an int64 slice.
func (p StringSlice) ToInt64() ([]int64, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]int64{})
	if err != nil {
		return nil, err
	}
	return val.([]int64), nil
}

// ToUint converts values to a uint slice.
func (p StringSlice) ToUint() ([]uint, error) {
	if p == nil {
		return nil, nil
	}
	ret := make([]uint, 0, len(p))
	for _, item := range p {
		if val64, err := strconv.ParseUint(item, 10, 64); err == nil {
			ret = append(ret, uint(val64))
		} else {
			return nil, err
		}
	}
	return ret, nil
}

// ToUint8 converts values to a uint8 slice.
func (p StringSlice) ToUint8() ([]uint8, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]uint8{})
	if err != nil {
		return nil, err
	}
	return val.([]uint8), nil
}

// ToUint16 converts values to a uint16 slice.
func (p StringSlice) ToUint16() ([]uint16, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]uint16{})
	if err != nil {
		return nil, err
	}
	return val.([]uint16), nil
}

// ToUint32 converts values to a uint32 slice.
func (p StringSlice) ToUint32() ([]uint32, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]uint32{})
	if err != nil {
		return nil, err
	}
	return val.([]uint32), nil
}

// ToUint64 converts values to a uint64 slice.
func (p StringSlice) ToUint64() ([]uint64, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]uint64{})
	if err != nil {
		return nil, err
	}
	return val.([]uint64), nil
}

// ToFloat32 converts values to a float32 slice.
func (p StringSlice) ToFloat32() ([]float32, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]float32{})
	if err != nil {
		return nil, err
	}
	return val.([]float32), nil
}

// ToFloat64 converts values to a float64 slice.
func (p StringSlice) ToFloat64() ([]float64, error) {
	if p == nil {
		return nil, nil
	}
	val, err := p.ToNumber([]float64{})
	if err != nil {
		return nil, err
	}
	return val.([]float64), nil
}

// ToInterface converts values to an interface slice.
func (p StringSlice) ToInterface() []interface{} {
	if len(p) == 0 {
		return nil
	}
	val := make([]interface{}, 0, len(p))
	for _, item := range p {
		val = append(val, item)
	}
	return val
}

// Int64 converts a value to int64.
func Int64(p interface{}) (i int64, err error) {
	switch v := p.(type) {
	case int:
		i = int64(v)
	case int8:
		i = int64(v)
	case int16:
		i = int64(v)
	case int32:
		i = int64(v)
	case int64:
		i = int64(v)
	case float32:
		i = int64(v)
	case float64:
		i = int64(v)
	case string:
		i, err = strconv.ParseInt(v, 10, 64)
	case json.Number:
		i, err = v.Int64()
	default:
		err = fmt.Errorf("unsupported type %T", p)
	}
	return
}

// Int converts a value to int.
func Int(p interface{}) (i int, err error) {
	switch v := p.(type) {
	case int:
		i = int(v)
	case int8:
		i = int(v)
	case int16:
		i = int(v)
	case int32:
		i = int(v)
	case int64:
		i = int(v)
	case float32:
		i = int(v)
	case float64:
		i = int(v)
	case string:
		var i64 int64
		i64, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		i = int(i64)
	case json.Number:
		var i64 int64
		i64, err = v.Int64()
		if err != nil {
			return
		}
		i = int(i64)
	default:
		err = fmt.Errorf("unsupported type %T", p)
	}
	return
}

// Float64 converts a value to float64.
func Float64(p interface{}) (i float64, err error) {
	switch v := p.(type) {
	case int:
		i = float64(v)
	case int8:
		i = float64(v)
	case int16:
		i = float64(v)
	case int32:
		i = float64(v)
	case int64:
		i = float64(v)
	case float32:
		i = float64(v)
	case float64:
		i = float64(v)
	case string:
		i, err = strconv.ParseFloat(v, 64)
	case json.Number:
		i, err = v.Float64()
	default:
		err = fmt.Errorf("unsupported type %T", p)
	}
	return
}

// Float32 converts a value to float32.
func Float32(p interface{}) (i float32, err error) {
	switch v := p.(type) {
	case int:
		i = float32(v)
	case int8:
		i = float32(v)
	case int16:
		i = float32(v)
	case int32:
		i = float32(v)
	case int64:
		i = float32(v)
	case float32:
		i = float32(v)
	case float64:
		i = float32(v)
	case string:
		var f64 float64
		f64, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return
		}
		i = float32(f64)
	case json.Number:
		var f64 float64
		f64, err = v.Float64()
		if err != nil {
			return
		}
		i = float32(f64)
	default:
		err = fmt.Errorf("unsupported type %T", p)
	}
	return
}

// IsEmpty reports whether any of the strings is empty.
func IsEmpty(strs ...string) bool {
	for _, str := range strs {
		if str == "" {
			return true
		}
	}
	return false
}

// HasNil check does any args is nil
func HasNil(args ...interface{}) bool {
	for _, arg := range args {
		if arg == nil {
			return true
		}
		var val = reflect.ValueOf(arg)
		var k = val.Kind()
		if (k == reflect.Ptr || k == reflect.Chan || k == reflect.Func || k == reflect.Map || k == reflect.Interface || k == reflect.Slice) &&
			val.IsNil() {
			return true
		}
	}
	return false
}

// Fnv32Hashcode calculate abs hash code for data
func Fnv32Hashcode(data string) int {
	hash := fnv.New32a()
	hash.Write([]byte(data))
	hashCode := int(hash.Sum32())
	if hashCode < 0 {
		hashCode = -hashCode
	}
	return hashCode
}

// MGet get value from m
func MGet(m map[string]interface{}, key string) interface{} {
	if val, ok := m[key]; ok {
		return val
	}
	return nil
}

// MGetInt32 get int32 from m
func MGetInt32(m map[string]interface{}, key string) (int32, error) {
	data := MGet(m, key)
	switch inst := data.(type) {
	case int:
		return int32(inst), nil
	case int8:
		return int32(inst), nil
	case int16:
		return int32(inst), nil
	case int32:
		return int32(inst), nil
	case int64:
		return int32(inst), nil
	case float32:
		return int32(inst), nil
	case float64:
		return int32(data.(float64)), nil
	case json.Number:
		v, err := inst.Int64()
		if err != nil {
			return 0, err
		}
		return int32(v), nil
	default:
		return 0, fmt.Errorf("invalid int32 type:%T", inst)
	}
}

// MGetInt64 get int64 from m
func MGetInt64(m map[string]interface{}, key string) (int64, error) {
	data := MGet(m, key)
	switch inst := data.(type) {
	case int:
		return int64(inst), nil
	case int8:
		return int64(inst), nil
	case int16:
		return int64(inst), nil
	case int32:
		return int64(inst), nil
	case int64:
		return int64(inst), nil
	case float32:
		return int64(inst), nil
	case float64:
		return int64(inst), nil
	case json.Number:
		return inst.Int64()
	default:
		return 0, fmt.Errorf("invalid int64 type:%T", inst)
	}
}

// MGetString get string from m
func MGetString(m map[string]interface{}, key string) (string, error) {
	data := MGet(m, key)
	switch data.(type) {
	case string:
		return data.(string), nil
	}
	return "", errors.New("invalid string type")
}

// FillSlice create slice
func FillSlice(num int, fillFunc func(index int)) {
	for i := 0; i < num; i++ {
		fillFunc(i)
	}
}

// ToSlice to slice
func ToSlice(num int, fillFunc func(index int, dest []interface{})) []interface{} {
	var ret = make([]interface{}, num)
	for i := 0; i < num; i++ {
		fillFunc(i, ret)
	}
	return ret
}

// ByteSlice2String convert []byte to string
func ByteSlice2String(bs []byte) (str string) {
	return string(str)
}

// String2ByteSlice convert string to []byte
func String2ByteSlice(str string) (bs []byte) {
	return []byte(str)
}

// StackTrace record the stack trace
func StackTrace(all bool) string {
	// Reserve 10K buffer at first
	buf := make([]byte, 10240)
	for {
		size := runtime.Stack(buf, all)
		// The size of the buffer may be not enough to hold the stacktrace,
		// so double the buffer size
		if size == len(buf) {
			buf = make([]byte, len(buf)<<1)
			continue
		}
		break
	}
	return string(buf)
}

// StructCopier copies values between structs.
type StructCopier func(from interface{}, to interface{}) (err error)

// NewStructCopier creates a struct copier.
func NewStructCopier(from interface{}, to interface{}) (copier StructCopier, err error) {
	if from == nil || to == nil {
		err = errors.New("from and to must not be nil")
		return
	}

	fromVal, fromInd, fromTyp := ExtractRefTuple(from)
	toVal, toInd, toTyp := ExtractRefTuple(to)

	if fromVal.Kind() != reflect.Ptr || toVal.Kind() != reflect.Ptr {
		err = errors.New("from and to must be pointer")
		return
	}
	if fromInd.Kind() != reflect.Struct || toInd.Kind() != reflect.Struct {
		err = errors.New("from and to must be struct")
		return
	}

	var noCopy = func(field reflect.StructField) bool {
		tag := field.Tag
		_, exist := tag.Lookup("nocopy")
		if exist {
			return true
		}
		return false
	}

	var (
		fromIndexes, toIndexes [][]int
	)

	var parseFields func(typ reflect.Type, baseFieldIndex []int) (err error)
	parseFields = func(typ reflect.Type, baseFieldIndex []int) (err error) {
		fieldCount := typ.NumField()
		for i := 0; i < fieldCount; i++ {
			field := typ.Field(i)
			if field.Anonymous {
				newIndex := make([]int, len(baseFieldIndex))
				copy(newIndex, baseFieldIndex)
				err = parseFields(field.Type, append(newIndex, field.Index...))
				if err != nil {
					return
				}
				continue
			} else if field.PkgPath != "" {
				continue
			}
			if noCopy(field) {
				continue
			}
			name := field.Name
			toField, found := toTyp.FieldByName(name)
			if !found {
				continue
			}
			if noCopy(toField) {
				continue
			}

			newIndex := make([]int, len(baseFieldIndex))
			copy(newIndex, baseFieldIndex)
			fromIndex := append(newIndex, field.Index...)
			toIndex := toField.Index

			var typeMatch bool
			if field.Type.AssignableTo(toField.Type) {
				typeMatch = true
			} else if field.Type.Kind() == toField.Type.Kind() && field.Type.ConvertibleTo(toField.Type) {
				typeMatch = true
			}
			if !typeMatch {
				err = fmt.Errorf("name %s can't assign %s to %s", name, field.Type, toField.Type)
				return
			}
			fromIndexes = append(fromIndexes, fromIndex)
			toIndexes = append(toIndexes, toIndex)
		}
		return
	}

	err = parseFields(fromTyp, nil)
	if err != nil {
		return
	}

	var (
		fromValType = fromVal.Type()
		fromValKind = fromVal.Kind()
		toValType   = toVal.Type()
		toValKind   = toVal.Kind()
	)
	copier = func(f, t interface{}) error {
		if f == nil || t == nil {
			return errors.New("from and to must not be nil")
		}

		fval, find, ftype := ExtractRefTuple(f)
		tval, tind, ttype := ExtractRefTuple(t)
		if ftype != fromTyp || fval.Kind() != fromValKind {
			return fmt.Errorf("expect the type of from is %s,but is %s", fromValType, fval.Type())
		}
		if ttype != toTyp || tval.Kind() != toValKind {
			return fmt.Errorf("expect the type of to is %s,but is %s", toValType, tval.Type())
		}

		for i := 0; i < len(fromIndexes); i++ {
			fromVal := find.FieldByIndex(fromIndexes[i])
			toVal := tind.FieldByIndex(toIndexes[i])
			if fromVal.Type() != toVal.Type() {
				toVal.Set(fromVal.Convert(toVal.Type()))
			} else {
				toVal.Set(fromVal)
			}
		}
		return nil
	}
	return
}

type emptyInterface struct {
	typ  *struct{}
	word *struct{}
}

type emptySliceInterface struct {
	typ  *struct{}
	word *reflect.SliceHeader
}

// IsValNil reports whether v is nil.
func IsValNil(v interface{}) bool {
	if v == nil {
		return true
	}
	typ := reflect.TypeOf(v)
	kind := typ.Kind()
	if kind == reflect.Ptr {
		ei := (*emptyInterface)(unsafe.Pointer(&v))
		return ei.word == nil
	} else if kind == reflect.Slice {
		ei := (*emptySliceInterface)(unsafe.Pointer(&v))
		return ei.word.Data == 0
	} else {
		switch kind {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Interface:
			return reflect.ValueOf(v).IsNil()
		}
	}
	return false
}

// IntSlice int slice
type IntSlice []int

// ToInterface converts values to an interface slice.
func (p IntSlice) ToInterface() []interface{} {
	if len(p) == 0 {
		return nil
	}
	val := make([]interface{}, 0, len(p))
	for _, item := range p {
		val = append(val, item)
	}
	return val
}

func (p IntSlice) Len() int           { return len(p) }
func (p IntSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p IntSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Int64Slice int64 slice
type Int64Slice []int64

// ToInterface converts values to an interface slice.
func (p Int64Slice) ToInterface() []interface{} {
	if len(p) == 0 {
		return nil
	}
	val := make([]interface{}, 0, len(p))
	for _, item := range p {
		val = append(val, item)
	}
	return val
}

func (p Int64Slice) Len() int           { return len(p) }
func (p Int64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Int32Slice int32 slice
type Int32Slice []int32

// ToInterface converts values to an interface slice.
func (p Int32Slice) ToInterface() []interface{} {
	if len(p) == 0 {
		return nil
	}
	val := make([]interface{}, 0, len(p))
	for _, item := range p {
		val = append(val, item)
	}
	return val
}

func (p Int32Slice) Len() int           { return len(p) }
func (p Int32Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p Int32Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
