package common

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// ExtractRefTuple 抽取反射的val:ValueOf ,ind:Indirect,typ:ind.Type
func ExtractRefTuple(obj interface{}) (val reflect.Value, ind reflect.Value, typ reflect.Type) {
	val = reflect.ValueOf(obj)
	ind = reflect.Indirect(val)
	typ = ind.Type()
	return
}

func getFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	val := reflect.Indirect(reflect.ValueOf(structObj))
	Debugf("val:%v", reflect.Indirect(val))
	return val.Field(fieldIndex).Type()
}

// GetFieldType 取得structObje的指定字段的类型
func GetFieldType(structObj interface{}, fieldIndex int) reflect.Type {
	return getFieldType(structObj, fieldIndex)
}

// GetFirstFieldType 取得structObj的第一个字段的类型
func GetFirstFieldType(structObj interface{}) reflect.Type {
	return getFieldType(structObj, 0)
}

// ReadLineWithProcessor 按行读取数据,没读取到一行就调用processorFunc进行处理,如果processorFunc返回false,
// 则停止读取并返回
func ReadLineWithProcessor(rd io.Reader, processorFunc func(line string) bool) error {
	r := bufio.NewReaderSize(rd, 4*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		if !processorFunc(s) {
			return nil
		}
		line, isPrefix, err = r.ReadLine()
	}
	if err != io.EOF {
		return err
	}
	return nil
}

// Shutdownhook 停止hook
type Shutdownhook struct {
	ch         chan os.Signal //接收信号的channel
	hooks      []func()       //停机时需要调用的方法列表
	sync.Mutex                //同步锁
}

// NewShutdownhook 创建一个Shutdownhook,sig是要监听的信号,默认会监听syscall.SIGINT,syscall.SIGTERM
func NewShutdownhook(sig ...os.Signal) *Shutdownhook {
	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	ch := make(chan os.Signal, len(sig))
	signal.Notify(ch, sig...)
	return &Shutdownhook{ch: ch}
}

// AddHook 增加一个Hook函数
func (p *Shutdownhook) AddHook(hookFunc func()) {
	p.Lock()
	defer p.Unlock()
	p.hooks = append(p.hooks, hookFunc)
}

// WaitShutdown 等待进程退出的信号,当收到进程退出的信号后,依次执行注册的hook函数
func (p *Shutdownhook) WaitShutdown() {
	p.Lock()
	defer p.Unlock()

	if p.ch == nil {
		panic("singal channel is nil")
	}

	if s, ok := <-p.ch; ok {
		signal.Stop(p.ch)
		close(p.ch)
		p.ch = nil

		Infof("Receive signal:%v,Run hooks", s)
		for _, f := range p.hooks {
			f()
		}
		Infof("Finished run hooks")
	} else {
		Warnf("Receive signal error,%v", ok)
	}
}

// RandomUUID 生成随机的UUID
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

// FileBasename 确定不待扩展的文件名
func FileBasename(s string) string {
	n := strings.LastIndexByte(s, '.')
	if n > 0 {
		return s[:n]
	}
	return s
}

// SplitTrimOmitEmpty 对str按sep分隔,去掉为空的项
func SplitTrimOmitEmpty(str, sep string) []string {
	return TrimOmitEmpty(strings.Split(str, sep))
}

// TrimOmitEmpty 去掉为空的值
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

// ToInt 转为[]int
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

//ToNumber 转为数字slice
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

// ToInt32 转为[]int32
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

// ToInt64 转为[]int64
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

// ToFloat32 转为[]float32
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

// ToFloat64 转为[]floa64t
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
