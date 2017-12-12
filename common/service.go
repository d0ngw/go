package common

import (
	"reflect"
	"sort"
	"sync"
)

// ServiceState 表示服务的状态
type ServiceState uint32

const (
	// NEW 新建``
	NEW ServiceState = iota
	// INITED 初始化完毕
	INITED
	// STARTING 正在启动
	STARTING
	// RUNNING 正在运行
	RUNNING
	// STOPPING 正在停止
	STOPPING
	// TERMINATED 已经停止
	TERMINATED
	// FAILED 失败
	FAILED
)

var serviceStateStrings = map[ServiceState]string{
	NEW:        "NEW",
	INITED:     "INITED",
	STARTING:   "STARTING",
	RUNNING:    "RUNNING",
	STOPPING:   "STOPPING",
	TERMINATED: "TERMINATED",
	FAILED:     "FAILED"}

func (p ServiceState) String() string {
	return serviceStateStrings[p]
}

var validStateState = map[ServiceState][]ServiceState{
	NEW:        []ServiceState{INITED, FAILED, TERMINATED},
	INITED:     []ServiceState{STARTING, FAILED, TERMINATED},
	STARTING:   []ServiceState{RUNNING, FAILED, TERMINATED},
	RUNNING:    []ServiceState{STOPPING, FAILED, TERMINATED},
	STOPPING:   []ServiceState{TERMINATED, FAILED},
	TERMINATED: []ServiceState{},
	FAILED:     []ServiceState{},
}

// IsValidServiceState 检查ServiceState的状态转移是否有效
func IsValidServiceState(oldState ServiceState, newState ServiceState) bool {
	if targetStates, ok := validStateState[oldState]; ok == true {
		for _, targetState := range targetStates {
			if targetState == newState {
				return true
			}
		}
	}
	return false
}

// Initable 表示需要进行初始化
type Initable interface {
	// Init 执行初始化操作,如果初始化失败,返回错误的原因
	Init() error
}

// Service 统一的服务接口
type Service interface {
	Initable
	// Name 取得服务名称
	Name() string
	// Start 启动服务
	Start() bool
	// Stop 停止服务
	Stop() bool
	// State 服务的状态
	State() ServiceState
	// SetState 设置服务的状态
	setState(newState ServiceState) bool
}

// ServiceInit 初始化服务
func ServiceInit(service Service) bool {
	Debugf("Init %T#%s", service, service.Name())
	if service.State() == INITED {
		Infof("%T#%s has been inited,skip", service, service.Name())
		return true
	}
	Infof("Init %T#%s", service, service.Name())
	err := service.Init()
	if err == nil && service.setState(INITED) {
		Debugf("Init %T#%s succ", service, service.Name())
		return true
	}
	Infof("Init %T#%s fail,err:%s", service, service.Name(), err)
	service.setState(FAILED)
	return false
}

// ServiceStart 开始服务
func ServiceStart(service Service) bool {
	Infof("Start %T#%s,state:%s", service, service.Name(), service.State())
	service.setState(STARTING)
	if service.Start() && service.setState(RUNNING) {
		Debugf("Start %T#%s succ", service, service.Name())
		return true
	}
	Infof("Start %T#%s fail", service, service.Name())
	service.setState(FAILED)
	return false
}

// ServiceStop 停止服务
func ServiceStop(service Service) bool {
	Debugf("Stop %T#%s", service, service.Name())
	service.setState(STOPPING)
	if service.Stop() && service.setState(TERMINATED) {
		Debugf("Stop %T#%s succ", service, service.Name())
		return true
	}
	Infof("Stop %T#%s fail", service, service.Name())
	service.setState(FAILED)
	return false
}

// BaseService 提供基本的Service接口实现
type BaseService struct {
	SName     string       //服务的名称
	state     ServiceState //服务的状态
	stateLock sync.RWMutex //读写锁
}

// Name 服务名称
func (p *BaseService) Name() string {
	return p.SName
}

// Init 初始化
func (p *BaseService) Init() error {
	return nil
}

// Start 启动服务
func (p *BaseService) Start() bool {
	return true
}

// Stop 停止服务
func (p *BaseService) Stop() bool {
	return true
}

// State 取得服务的状态
func (p *BaseService) State() ServiceState {
	p.stateLock.RLock()
	defer p.stateLock.RUnlock()
	return p.state
}

func (p *BaseService) setState(newState ServiceState) bool {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()
	if IsValidServiceState(p.state, newState) {
		p.state = newState
		return true
	}
	Criticalf("Invalid state transfer %s->%s,%s", p.state, newState, p.Name())
	return false
}

// ServiceSorter Service的排序
type ServiceSorter func(servies []Service) sort.Interface

// Services 一组Service的集合
type Services struct {
	sorted []Service     //排序后的服务集合
	sorter ServiceSorter //服务的排序
}

// NewServices 构建新的Service集合
func NewServices(services []Service, serviceSorter ServiceSorter) *Services {
	//排序
	var sorted []Service
	if serviceSorter != nil {
		t := serviceSorter(services)
		sort.Sort(t)
		tv := reflect.ValueOf(t)
		a := make([]Service, 0, len(services))
		for i := 0; i < len(services); i++ {
			a = append(a, tv.Index(i).Interface().(Service))
		}
		sorted = a
	} else {
		sorted = services
	}
	return &Services{sorted: sorted, sorter: serviceSorter}
}

// Init 初始化服务集合
func (p *Services) Init() bool {
	for _, service := range p.sorted {
		if !ServiceInit(service) {
			Warnf("Init service %T#%s fail", service, service.Name())
			return false
		}
	}
	return true
}

// Start 启动服务
func (p *Services) Start() bool {
	for _, service := range p.sorted {
		Infof("Start service %T#%s", service, service.Name())
		if !ServiceStart(service) {
			Warnf("Start service %T#%s fail", service, service.Name())
			return false
		}
	}
	return true
}

// Stop 停止服务
func (p *Services) Stop() bool {
	for i := len(p.sorted) - 1; i >= 0; i-- {
		service := p.sorted[i]
		Infof("Stop service %T#%s", service, service.Name())
		if !ServiceStop(service) {
			Warnf("Stop service %T#%s fail", service, service.Name())
		}
	}
	return true
}
