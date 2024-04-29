package argparse

import (
	"flag"
	"os"
	"strings"

	"github.com/kitex-contrib/kitexcall/pkg/config"
	"github.com/kitex-contrib/kitexcall/pkg/errors"
	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type Augment struct {
	config.Config
}

type EndpointList []string

func (e *EndpointList) String() string {
	return strings.Join(*e, ",")
}

func (e *EndpointList) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (a *Augment) buildFlags() *flag.FlagSet {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	f.StringVar(&a.Type, "type", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'.")
	f.StringVar(&a.Type, "t", "unknown", "Specify the type of IDL: 'thrift' or 'protobuf'. (shorthand)")

	f.StringVar(&a.IDLPath, "idl-path", "", "Specify the path of IDL file.")
	f.StringVar(&a.IDLPath, "p", "", "Specify the path of IDL file. (shorthand)")

	f.StringVar(&a.Service, "service", "", "Specify the service name.")
	f.StringVar(&a.Service, "s", "", "Specify the service name. (shorthand)")

	f.StringVar(&a.Method, "method", "", "Specify the method name.")
	f.StringVar(&a.Method, "m", "", "Specify the method name. (shorthand)")

	f.StringVar(&a.Data, "data", "", "Specify the data to be sent.")
	f.StringVar(&a.Data, "d", "", "Specify the data to be sent.")

	f.Var((*EndpointList)(&a.Endpoint), "endpoint", "Specify the server endpoints. Can be repeated.")
	f.Var((*EndpointList)(&a.Endpoint), "e", "Specify the server endpoints. Can be repeated. (shorthand)")

	f.BoolVar(&a.Verbose, "verbose", false, "Enable verbose mode.")
	f.BoolVar(&a.Verbose, "v", false, "Enable verbose mode. (shorthand)")

	f.StringVar(&a.Transport, "transport", "", "Specify the transport type. Can be 'TTHeader', 'Framed', 'TTHeaderFramed'.")

	return f
}

func (a *Augment) ParseArgs() error {
	f := a.buildFlags()
	if err := f.Parse(os.Args[1:]); err != nil {
		return errors.New(errors.ArgParseError, "Flag parse error: %v", err)
	}
	log.Verbose = a.Verbose
	log.Info("Parse args succeed")

	if err := a.ValidateArgs(); err != nil {
		return err
	}
	return nil
}

func (a *Augment) ValidateArgs() error {
	// Since server reflection is not available yet,
	// this is the only way to obtain IDL.
	if a.IDLPath == "" {
		return errors.New(errors.ArgParseError, "IDL path is required")
	}

	if a.Type != "thrift" && a.Type != "protobuf" {
		return errors.New(errors.ArgParseError, "Invalid IDL type: %v", a.Type)
	}

	// This can be modified when service discovery is supported.
	if len(a.Endpoint) == 0 {
		return errors.New(errors.ArgParseError, "At least one endpoint must be specified")
	}

	if a.Service == "" {
		return errors.New(errors.ArgParseError, "Service name is required")
	}

	if a.Data == "" {
		return errors.New(errors.ArgParseError, "Data is required")
	}

	if a.Method == "" {
		return errors.New(errors.ArgParseError, "Method name is required")
	}

	if a.Transport != config.TTHeader && a.Transport != config.Framed && a.Transport != config.TTHeaderFramed && a.Transport != "" {
		return errors.New(errors.ArgParseError, "Transport type is invalid")
	}

	log.Info("Args validate succeed")
	return nil
}

func (a *Augment) BuildConfig() *config.Config {
	return &config.Config{
		Method:    a.Method,
		Verbose:   a.Verbose,
		Type:      a.Type,
		IDLPath:   a.IDLPath,
		Endpoint:  a.Endpoint,
		Service:   a.Service,
		Data:      a.Data,
		Transport: a.Transport,
	}
}
