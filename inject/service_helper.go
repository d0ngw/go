package inject

import (
	"fmt"
	c "github.com/d0ngw/go/common"
)

// Init 初始化
func (p *Injector) Init() error {
	// 执行初始化操作
	inits := p.GetInstancesByPrototype(struct{ s c.Initable }{})
	for _, init := range inits {
		if service, ok := init.(c.Service); ok {
			if !c.ServiceInit(service) {
				return fmt.Errorf("Init %s fail", service.Name())
			}
		} else {
			if err := init.(c.Initable).Init(); err != nil {
				return err
			}

		}
	}
	return nil
}

func (p *Injector) startOrStop(sorter c.ServiceSorter, start bool) error {
	var services []c.Service
	for _, service := range p.GetInstancesByPrototype(struct{ s c.Service }{}) {
		services = append(services, service.(c.Service))
	}
	if len(services) == 0 {
		c.Infof("No service found,skip")
		return nil
	}

	sortedServices := c.NewServices(services, sorter)
	if start {
		if !sortedServices.Start() {
			return fmt.Errorf("Start servcie fail")
		}
	} else {
		if !sortedServices.Stop() {
			return fmt.Errorf("Stop servcie fail")
		}
	}
	return nil
}

// Start 启动服务
func (p *Injector) Start(sorter c.ServiceSorter) error {
	return p.startOrStop(sorter, true)
}

// Stop 停止服务
func (p *Injector) Stop(sorter c.ServiceSorter) error {
	return p.startOrStop(sorter, false)
}
