package common

import (
	"bufio"
	"io"
	"os"
	"os/signal"
	"reflect"
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

// Shutdownhook
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
func (self *Shutdownhook) AddHook(hookFunc func()) {
	self.Lock()
	defer self.Unlock()
	self.hooks = append(self.hooks, hookFunc)
}

// WaitShutdown 等待进程退出的信号,当收到进程退出的信号后,依次执行注册的hook函数
func (self *Shutdownhook) WaitShutdown() {
	self.Lock()
	defer self.Unlock()

	if self.ch == nil {
		panic("singal channel is nil")
	}

	if s, ok := <-self.ch; ok {
		signal.Stop(self.ch)
		close(self.ch)
		self.ch = nil

		Infof("Receive signal:%v,Run hooks", s)
		for _, f := range self.hooks {
			f()
		}
		Infof("Finished run hooks")
	} else {
		Warnf("Receive signal error,%v", ok)
	}
}
