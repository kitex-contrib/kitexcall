package config

// Define constants for supported IDL types.
const (
	Unknown  string = ""
	Thrift   string = "thrift"
	Protobuf string = "protobuf"
	// Reserved for Server Streaming
)

// We provide a general configuration
// so that it can be utilized by others apart from kitexcall.
type Config struct {
	Verbose  bool
	Type     string
	IDLPath  string
	Endpoint []string
	Service  string
	Method   string
}

type ConfigParser interface {
	BuildConfig() (Config, error)
}
