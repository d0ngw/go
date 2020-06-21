package common

import (
	"fmt"
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
	NEW:        {INITED, FAILED, TERMINATED},
	INITED:     {STARTING, FAILED, TERMINATED},
	STARTING:   {RUNNING, FAILED, TERMINATED},
	RUNNING:    {STOPPING, FAILED, TERMINATED},
	STOPPING:   {TERMINATED, FAILED},
	TERMINATED: {},
	FAILED:     {},
}

// IsValidServiceState 检查ServiceState的状态转移是否有效
func IsValidServiceState(oldState ServiceState, newState ServiceState) bool {
	if targetStates, ok := validStateState[oldState]; ok {
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
	// 启动的次序
	GetStartOrder() int
	// Stop 停止服务
	Stop() bool
	// 停止的次序
	GetStopOrder() int
	// State 服务的状态
	State() ServiceState
	// SetState 设置服务的状态
	setState(newState ServiceState) bool
}

// ServiceInit 初始化服务
func ServiceInit(service Service) bool {
	if service.State() == INITED {
		Infof("%s has been inited,skip", service)
		return true
	}
	name := ServiceName(service)
	err := service.Init()
	if err == nil && service.setState(INITED) {
		return true
	}
	Errorf("init %s fail,err:%s", name, err)
	service.setState(FAILED)
	return false
}

// ServiceStart 开始服务
func ServiceStart(service Service) bool {
	name := ServiceName(service)
	service.setState(STARTING)
	if service.Start() && service.setState(RUNNING) {
		return true
	}
	Errorf("start %s fail", name)
	service.setState(FAILED)
	return false
}

// ServiceStop 停止服务
func ServiceStop(service Service) bool {
	name := ServiceName(service)
	service.setState(STOPPING)
	if service.Stop() && service.setState(TERMINATED) {
		return true
	}
	Errorf("stop %s fail", name)
	service.setState(FAILED)
	return false
}

// BaseService 提供基本的Service接口实现
type BaseService struct {
	SName     string //服务的名称
	Order     int
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

// GetStartOrder 启动服务
func (p *BaseService) GetStartOrder() int {
	return p.Order
}

// Stop 停止服务
func (p *BaseService) Stop() bool {
	return true
}

// GetStopOrder 停止服务
func (p *BaseService) GetStopOrder() int {
	return -p.GetStartOrder()
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

// ServiceName 取得服务的名称
func ServiceName(service Service) string {
	name := fmt.Sprintf("%T", service)
	if service.Name() != "" {
		name += "#" + service.Name()
	}
	return name
}

// Services 一组Service的集合
type Services struct {
	sorted []Service //排序后的服务集合
}

// NewServices 构建新的Service集合
func NewServices(services []Service, start bool) *Services {
	//排序
	var sorted = make([]Service, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool {
		if start {
			return sorted[i].GetStartOrder() < sorted[j].GetStartOrder()
		}
		return sorted[i].GetStopOrder() < sorted[j].GetStopOrder()
	})
	return &Services{sorted: sorted}
}

// Init 初始化服务集合
func (p *Services) Init() bool {
	for _, service := range p.sorted {
		if !ServiceInit(service) {
			Warnf("init %s fail", service)
			return false
		}
	}
	return true
}

// Start 启动服务
func (p *Services) Start() bool {
	for _, service := range p.sorted {
		name := ServiceName(service)
		if !ServiceStart(service) {
			Warnf("start %s fail", name)
			return false
		}
	}
	return true
}

// Stop 停止服务
func (p *Services) Stop() bool {
	for _, service := range p.sorted {
		name := ServiceName(service)
		if !ServiceStop(service) {
			Warnf("stop %s fail", name)
		}
	}
	return true
}
