package argparse

import (
	"flag"
	"fmt"
	"os"

	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type Augment struct {
	config.Config
}

func (a *Augment) buildFlags() *flag.FlagSet {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	f.StringVar(&a.Type, "type", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'.")
	f.StringVar(&a.Type, "t", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'. (shorthand)")

	f.StringVar(&a.IDLPath, "idl-path", "", "Specify the path of IDL file.")
	f.StringVar(&a.IDLPath, "p", "", "Specify the path of IDL file. (shorthand)")

	f.StringVar(&a.Service, "service", "", "Specify the service name.")
	f.StringVar(&a.Service, "s", "", "Specify the service name. (shorthand)")
	return f
}

func (a *Augment) ParseArgs() {
	fmt.Println("Parse Args")
	f := a.buildFlags()
	if err := f.Parse(os.Args[1:]); err != nil {
		log.Error("ArgParse", err)
		os.Exit(2)
	}

	fmt.Println("Validata Args")

	fmt.Println(f.Args()) // 这个是获取未被解析的参数列表。。。
	a.ValidateArgs()
}

func (a *Augment) ValidateArgs() {
	// 检查必需参数是否提供
	if a.IDLPath == "" {
		log.Error("ArgParse", "IDL path is required")
		os.Exit(2)
	}

	// 检查参数值的有效性
	if a.Type != "thrift" && a.Type != "protobuf" {
		log.Error("ArgParse", "Invalid IDL type: ", a.Type)
		os.Exit(2)
	}

	// 其他参数验证...

	fmt.Println("Args validation passed")
}

func (a *Augment) BuildConfig() (config.Config, error) {
	return config.Config{
		Method:   a.Method,
		Verbose:  false,
		Type:     "thrift",
		IDLPath:  "./example_service.thrift",
		Endpoint: []string{"0.0.0.0:9999"},
		Service:  "destServiceName",
	}, nil
}
