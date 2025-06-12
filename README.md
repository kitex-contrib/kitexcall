# **kitexcall** (*This is a community driven project*)

English | [中文](README_cn.md)

Kitexcall is a command-line tool for sending JSON general requests using kitex, similar to how curl is used for HTTP.

## Features

- **Supports Thrift/Protobuf:** It supports IDL in Thrift/Protobuf formats.
- **Supports Multiple Transport Protocols:** It supports transport protocols like Buffered, TTHeader, Framed, and TTHeaderFramed, as well as gRPC for streaming calls.
- **Supports Streaming Calls:** It supports unary, client streaming, server streaming, and bidirectional streaming RPCs.
- **Supports Interactive Mode:** Automatically enters interactive mode when no input file or data is specified, allowing users to input request data in real-time.
- **Supports Common Client Options:** It allows specify common client options, such as client.WithHostPorts, etc.
- **Supports manual data input from the command line and local files:** Request data can be read from command line arguments or local files.
- **Supports Metadata Passing:** It supports sending transient keys (WithValue) and persistent keys (WithPersistentValue), and also supports receiving backward metadata (Backward) returned by the server.
- **Supports Receiving Business Custom Exceptions:** It can receive business custom exception error codes, error messages, and additional information.
- **Supports Multiple Output Formats:** By default, it outputs a human-friendly readable format, and plans to support parseable formats for better integration with other automation tools.

## Installation

```bash
go install github.com/kitex-contrib/kitexcall@latest
```

## Usage

### Basic Usage

When using the kitexcall tool, you need to specify several required arguments, including the path to the IDL file, the method name, and the data to be sent. Example:

- IDL file：

```thrift
// echo.thrift

namespace go test

struct Request {
    1: required string message,
}

struct Response {
    1: required string message,
}

service TestService {
    Response Echo (1: Request req) (streaming.mode="bidirectional"),
    Response EchoClient (1: Request req) (streaming.mode="client"),
    Response EchoServer (1: Request req) (streaming.mode="server"),

    Response EchoPingPong (1: Request req), // KitexThrift, non-streaming
}
```

- Creating a file input.json specifying JSON format request data:

```json
{
    "message": "hello"
}
```

- Server:

```go
// TestServiceImpl implements echo.TestService interface
type TestServiceImpl struct{}

// Echo implements bidirectional streaming
func (s *TestServiceImpl) Echo(stream echo.TestService_EchoServer) (err error) {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// Echo back the message
		resp := &echo.Response{
			Message: "server echo: " + req.Message,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// EchoClient implements client streaming
func (s *TestServiceImpl) EchoClient(stream echo.TestService_EchoClientServer) (err error) {
	var messageCount int
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client has finished sending
			resp := &echo.Response{
				Message: "server received " + strconv.Itoa(messageCount) + " messages",
			}
			return stream.SendAndClose(resp)
		}
		if err != nil {
			return err
		}
		messageCount++
	}
}

// EchoServer implements server streaming
func (s *TestServiceImpl) EchoServer(req *echo.Request, stream echo.TestService_EchoServerServer) (err error) {
	counter := 0
	for {
		resp := &echo.Response{
			Message: "server streaming response " + strconv.Itoa(counter) + " for request: " + req.Message,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		counter++
	}
}

// EchoPingPong implements traditional request-response
func (s *TestServiceImpl) EchoPingPong(ctx context.Context, req *echo.Request) (resp *echo.Response, err error) {
	return &echo.Response{
		Message: "server pong: " + req.Message,
	}, nil
}

func main() {
	svr := echo.NewServer(new(TestServiceImpl),
		server.WithMetaHandler(transmeta.ServerHTTP2Handler),
	)

	err := svr.Run()
	if err != nil {
		log.Println(err.Error())
	}
}
```

### Usage Examples

1. Normal Calls:

```bash
# Method 1: Using file input
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999 -f input.json
```
Output:
```
[Status]: Success
{
    "message": "server pong: hello"
}
```

```bash
# Method 2: Direct data input
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999 -d '{"message": "hello"}'
```
Output:
```
[Status]: Success
{
    "message": "server pong: hello"
}
```

```bash
# Method 3: Interactive input
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999
```
> {"message": "hello"}
[Status]: Success
{
    "message": "server pong: hello"
}
```

2. Streaming Calls:

```bash
# Client streaming (using JSONL file input)
cat > messages.jsonl << EOF
{"message": "hello 1"}
{"message": "hello 2"}
{"message": "hello 3"}
EOF

kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming -f messages.jsonl
```
Output:
```
[Status]: Success
{
    "message": "server received 3 messages"
}
```

```bash
# Server streaming (using JSON file input)
kitexcall --idl-path echo.thrift --method TestService/EchoServer --endpoint 127.0.0.1:8888 --streaming -f input.json
```
Output:
```
[Status]: Success
{
    "message": "server streaming response 0 for request: hello"
}
{
    "message": "server streaming response 1 for request: hello"
}
{
    "message": "server streaming response 2 for request: hello"
}
...
```

```bash
# Bidirectional streaming (using interactive input)
kitexcall --idl-path echo.thrift --method TestService/Echo --endpoint 127.0.0.1:8888 --streaming
```
> {"message": "hello"}
[Status]: Success
{
    "message": "server echo: hello"
}

# Use Ctrl+D to end input (no more requests will be sent, but streaming responses will still be received if the server continues to send)
# Use Ctrl+C to terminate the streaming session (force close the connection)
```

### Command Line Options

- `-help` or `-h`: Outputs the usage instructions.
- `-type` or `-t`: Specifies the IDL type: thrift or protobuf. It supports inference based on the IDL file type. The default is thrift..
- `-idl-path` or `-p`: Specifies the path to the IDL file.
- `-include-path`: Add a search path for the IDL. Multiple paths can be added and will be searched in the order they are added.
- `-method` or `-m`: Required, specifies the method name in the format IDLServiceName/MethodName or just MethodName. When the server side has MultiService mode enabled, IDLServiceName must be specified, and the transport protocol must be TTHeader or TTHeaderFramed.
- `-file` or `-f`: Specifies the input file path, which must be in JSON format.
- `-data` or `-d`: Specifies the data to be sent, in JSON string format.
- `-endpoint` or `-e`: Specifies the server address, multiple can be specified.
- `-transport`: Specifies the transport protocol type. It can be TTHeader, Framed, or TTHeaderFramed. If not specified, the default is Buffered.
- `-biz-error`: Enables the client to receive business errors returned by the server.
- `-meta`: Specifies one-way metadata passed to the server. Multiple can be specified, in the format key=value.
- `-meta-persistent`: Specifies persistent metadata passed to the server. Multiple can be specified, in the format key=value.
- `-meta-backward`: Enables receiving backward metadata (Backward) returned by the server.
- `-q`: Only output JSON response, no other information.
- `-verbose` or `-v`: Enables verbose mode for more detailed output information.

### Detailed Description

#### IDL Type

Use the `-type` or `-t` flag to specify the IDL type. Supported types are thrift and protobuf, with a default of thrift.

```bash
kitexcall -t thrift
```

#### IDL Path

Use the `-idl-path` or `-p` flag to specify the path to the IDL file.

```bash
kitexcall -idl-path /path/to/idl/file.thrift
```

#### Method to Call (Required)

Use the `-method` or `-m` flag to specify the method name. The format can be IDLServiceName/MethodName or just MethodName. When the server side has MultiService mode enabled, IDLServiceName must be specified, and the transport protocol must be TTHeader or TTHeaderFramed.

```bash
kitexcall -m GenericService/ExampleMethod
kitexcall -m ExampleMethod
```

#### Request Data

Use the `-data` or `-d` flag to specify the data to be sent, which should be a JSON formatted string. Alternatively, use the `-file` or `-f` flag to specify the path to a JSON file containing the data.

Assuming we want to send the data `{"message": "hello"}`, we can specify the data like this:

```bash
kitexcall -m ExampleMethod -d '{"message": "hello"}'
```

Or, we can save the data in a JSON file, such as input.json, and then specify the file path:

```bash
kitexcall -m ExampleMethod -f input.json
```

#### Server Address

Use the `-endpoint` or `-e` flag to specify one or more server addresses.

Assuming we have two server addresses, 127.0.0.1:9919 and 127.0.0.1:9920, we can specify the server addresses like this:

```bash
kitexcall -m ExampleMethod -e 127.0.0.1:9919 -e 127.0.0.1:9920
```

#### Metadata

- Transient keys (WithValue): Use the `-meta` flag to specify, multiple can be specified, in the format key=value.
- Persistent keys (WithPersistentValue): Use the `-meta-persistent` flag to specify. Multiple can be specified, in the format key=value.
- Backward metadata (Backward): Enable the `-meta-backward` option to support receiving backward metadata (Backward) returned by the server.

Assuming we want to pass Transient  metadata `temp=temp-value` and persistent metadata `logid=12345` to the server and receive backward metadata, we can specify it like this:

```bash
kitexcall -m ExampleMethod -meta temp=temp-value -meta-persistent logid=12345 -meta-backward
```

#### Business Exceptions

If the server returns a business exception, you can enable the client to receive business custom exception error codes, error messages, and additional information through the `-biz-error` flag.

Assuming the server returns a business error status code 404 and message `not found`, we can enable business error handling like this:

```bash
kitexcall -m ExampleMethod -biz-error
```

#### Enable Verbose Mode

Use the `-verbose` or `-v` flag to enable verbose mode, providing more detailed output information.

### Streaming Support

Kitexcall supports gRPC streaming RPC calls. When using streaming mode, the transport protocol is automatically set to gRPC.

#### Streaming Command Line Options

- `--streaming`: Enables streaming mode. The streaming type is automatically determined based on the method definition in the IDL file.

#### Streaming Input Files

For client and bidirectional streaming, which require sending multiple messages, you can use:
- A single JSON file (.json) for sending a single message
- A JSONL file (.jsonl) for sending multiple messages, with one JSON object per line

#### Examples

**Client Streaming Example:**

```bash
# Create a file with multiple messages (one per line)
cat > messages.jsonl << EOF
{"message": "hello 1"}
{"message": "hello 2"}
{"message": "hello 3"}
EOF

# Send multiple messages with client streaming
kitexcall -idl-path echo.thrift -m echo -f messages.jsonl --streaming
```

**Server Streaming Example:**

```bash
# Send a single request and receive multiple responses
kitexcall -idl-path echo.thrift -m echo -d '{"message": "hello"}' --streaming
```

**Bidirectional Streaming Example:**

```bash
# Send multiple messages and receive multiple responses
kitexcall -idl-path echo.thrift -m echo -f messages.jsonl --streaming
```

### Interactive Mode

When no input file (`-f`) or data (`-d`) is specified, kitexcall automatically enters interactive mode. In interactive mode, you can:

1. Input request data in real-time
2. View server responses
3. Continue inputting new request data
4. Use `Ctrl+D` to end input (no more requests will be sent, but streaming responses will still be received if the server continues to send)
5. Use `Ctrl+C` to terminate the streaming session (force close the connection)

Example:

```bash
# No input file or data specified, automatically enters interactive mode
kitexcall -idl-path echo.thrift -m echo

# Input data in interactive mode
> {"message": "hello"}
[Status]: Success
{
    "message": "hello"
}

> {"message": "world"}
[Status]: Success
{
    "message": "world"
}

# Use Ctrl+D to end input (no more requests will be sent, but streaming responses will still be received if the server continues to send)
# Use Ctrl+C to terminate the streaming session (force close the connection)
```

Interactive mode is particularly useful for:
- Scenarios requiring multiple requests with different data
- Debugging and testing service interfaces
- Real-time viewing of service responses
