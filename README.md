# go
Go Toolkit

## Inject example
```go
import (
	"flag"
	"fmt"
	"os"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/inject"
)

// WorkerApp woker服务,适用于后台执行的任务
type WorkerApp struct {
	config  c.Configurer
	modules []*inject.Module
	env     string
}

// NewWorkerApp new http app
func NewWorkerApp(config c.Configurer, modules []*inject.Module) *WorkerApp {
	ret := &WorkerApp{
		config:  config,
		modules: modules,
	}
	return ret
}

// Usage flag.Usage
func (p *WorkerApp) Usage() {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "usage:%s args...\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
	os.Exit(1)
}

// FlagInit flag的初始化
func (p *WorkerApp) FlagInit() {
	flag.Usage = p.Usage
	flag.StringVar(&p.env, "env", "dev", "the enviroment")
}

// Run 运行,直到推出
func (p *WorkerApp) Run() {
	flag.Parse()

	injector, err := inject.SetupInjector(p.config, "", p.env, p.modules...)
	if err != nil {
		c.Errorf("init injector fail,err:%s", err)
		p.Usage()
	}

	if err := injector.Start(nil); err != nil {
		c.Errorf("Start servcie fail,err:%s", err)
		os.Exit(1)
	}

	shutdownHook := c.NewShutdownhook(defaultShutdownHooks...)
	shutdownHook.AddHook(func() {
		injector.Stop(nil)
	})

	shutdownHook.WaitShutdown()
}


var app = NewWorkerApp(xxxconfig,xxxmodules)

func init(){
    app.FlagInit()
}

func main(){
    app.Run()
}
```
