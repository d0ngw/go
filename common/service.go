package common

import (
	"fmt"
	"sort"
	"sync"
)

// ServiceState represents a service lifecycle state.
type ServiceState uint32

const (
	// NEW indicates that the service has been created.
	NEW ServiceState = iota
	// INITED indicates that initialization completed.
	INITED
	// STARTING indicates that the service is starting.
	STARTING
	// RUNNING indicates that the service is running.
	RUNNING
	// STOPPING indicates that the service is stopping.
	STOPPING
	// TERMINATED indicates that the service has stopped.
	TERMINATED
	// FAILED indicates that a service operation failed.
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

// IsValidServiceState reports whether a service state transition is valid.
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

// Initable defines an instance that requires initialization.
type Initable interface {
	// Init initializes the instance and returns an error on failure.
	Init() error
}

// Service defines the service lifecycle interface.
type Service interface {
	Initable
	// Name returns the service name.
	Name() string
	// Start starts the services.
	Start() bool
	// GetStartOrder returns the startup order.
	GetStartOrder() int
	// Stop stops the services.
	Stop() bool
	// GetStopOrder returns the shutdown order.
	GetStopOrder() int
	// State returns the service state.
	State() ServiceState
	// setState updates the service state.
	setState(newState ServiceState) bool
}

// ServiceInit initializes a service and updates its state.
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

// ServiceStart starts a service and updates its state.
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

// ServiceStop stops a service and updates its state.
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

// BaseService provides a basic Service implementation.
type BaseService struct {
	SName     string // Service name.
	Order     int
	state     ServiceState // Service state.
	stateLock sync.RWMutex // Read/write lock.
}

// Name returns the service name.
func (p *BaseService) Name() string {
	return p.SName
}

// Init initializes the service.
func (p *BaseService) Init() error {
	return nil
}

// Start starts the services.
func (p *BaseService) Start() bool {
	return true
}

// GetStartOrder returns the service startup order.
func (p *BaseService) GetStartOrder() int {
	return p.Order
}

// Stop stops the services.
func (p *BaseService) Stop() bool {
	return true
}

// GetStopOrder returns the service shutdown order.
func (p *BaseService) GetStopOrder() int {
	return -p.GetStartOrder()
}

// State returns the service state.
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
	Errorf("Invalid state transfer %s->%s,%s", p.state, newState, p.Name())
	return false
}

// ServiceName returns the service type and optional name.
func ServiceName(service Service) string {
	name := fmt.Sprintf("%T", service)
	if service.Name() != "" {
		name += "#" + service.Name()
	}
	return name
}

// Services is an ordered collection of services.
type Services struct {
	sorted []Service // Sorted services.
}

// NewServices creates a service collection sorted for startup or shutdown.
func NewServices(services []Service, start bool) *Services {
	// Sort the services.
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

// Init initializes the services.
func (p *Services) Init() bool {
	for _, service := range p.sorted {
		if !ServiceInit(service) {
			Warnf("init %s fail", service)
			return false
		}
	}
	return true
}

// Start starts the services.
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

// Stop stops the services.
func (p *Services) Stop() bool {
	ok := true
	for _, service := range p.sorted {
		name := ServiceName(service)
		if !ServiceStop(service) {
			Warnf("stop %s fail", name)
			ok = false
		}
	}
	return ok
}
