# kitexcall (*这是一个社区驱动的项目*)

中文 | [English](README.md)

kitexcall 是使用 kitex 发送 json 通用请求的命令行工具，像是 curl 之于 HTTP。

## **Features**

- **支持 Thrift/Protobuf**：支持 Thrift/Protobuf 格式的 IDL。
- **支持多种传输协议**：支持 Buffered、TTHeader、Framed、TTHeaderFramed 传输协议，以及用于流式调用的 gRPC 协议。
- **支持流式调用**：支持单向、客户端流式、服务端流式和双向流式 RPC 调用。
- **支持交互式模式**：当未指定输入文件或数据时，自动进入交互式模式，方便用户实时输入请求数据。
- **支持常用客户端选项**：支持指定常用的客户端选项，例如 `client.WithHostPorts` 等。
- **支持手动从命令行和本地文件输入数据**：请求数据可以从命令行参数或本地文件读取。
- **支持元信息传递**：支持发送单跳透传（WithValue）和持续透传（WithPersistentValue）的元信息，并支持接收 server 返回的反向透传元信息（Backward）。
- **支持接收业务自定义异常**：接收业务自定义的异常错误码、错误信息及附加信息。
- **支持多种输出格式**：默认情况下输出人类友好的可读格式，计划支持可解析的格式，以便与其他自动化工具更好地集成。

## **Installation**

```bash
go install github.com/kitex-contrib/kitexcall@latest
```

## **Usage**

### 基本用法

使用 `kitexcall` 工具时，需要指定多个必选参数，包括 IDL 文件的路径、方法名以及要发送的数据。示例：

对应的 IDL 文件:

```python
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

创建input.json文件指定json格式请求数据：

```json
{
    "message": "hello"
}
```

对应的 server:

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

### 使用示例

1. 普通调用：

```bash
# 方式一：使用文件输入
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999 -f input.json
```
输出：
```
[Status]: Success
{
    "message": "server pong: hello"
}
```

```bash
# 方式二：直接指定数据
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999 -d '{"message": "hello"}'
```
输出：
```
[Status]: Success
{
    "message": "server pong: hello"
}
```

```bash
# 方式三：交互式输入
kitexcall --idl-path echo.thrift --method TestService/EchoPingPong --endpoint 127.0.0.1:9999

> {"message": "hello"}
[Status]: Success
{
    "message": "server pong: hello"
}
```

2. 流式调用：
```bash
# 客户端流式调用（使用JSONL文件输入）
cat > messages.jsonl << EOF
{"message": "hello 1"}
{"message": "hello 2"}
{"message": "hello 3"}
EOF

kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming -f messages.jsonl
```
输出：
```
[Status]: Success
{
    "message": "server received 3 messages"
}
```

```bash
# 服务端流式调用（使用JSON文件输入）
kitexcall --idl-path echo.thrift --method TestService/EchoServer --endpoint 127.0.0.1:8888 --streaming -f input.json
```
输出：
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
# 双向流式调用（使用交互式输入）
kitexcall --idl-path echo.thrift --method TestService/Echo --endpoint 127.0.0.1:8888 --streaming

> {"message": "hello"}
[Status]: Success
{
    "message": "server echo: hello"
}

# 使用 Ctrl+D 结束输入（不再发送新请求，但如果服务端继续推送，仍会接收流式响应）
# 使用 Ctrl+C 终止流式会话（强制关闭连接）
```

### 命令行选项

- `-help` 或 `-h`
  输出使用说明。
- `-type` 或 `-t`
  指定 IDL 类型：`thrift` 或 `protobuf`，支持通过 IDL 文件类型推测，默认是 `thrift`。
- `-idl-path` 或 `-p`
  指定 IDL 文件的路径。
- `-include-path`
  添加一个 IDL 里 include 的其他文件的搜索路径。支持添加多个，会按照添加的路径顺序搜索。
- `-method` 或 `-m`
  【必选】指定方法名，格式为 `IDLServiceName/MethodName` 或仅为 `MethodName`。当 server 端开启了 MultiService 模式时，必须指定 `IDLServiceName`，同时指定传输协议为 `TTHeader` 或 `TTHeaderFramed`。
- `-file` 或 `-f`
  指定输入文件路径，必须是 JSON 格式。
- `-data` 或 `-d`
  指定要发送的数据，格式为 JSON 字符串。
- `-endpoint` 或 `-e`
  指定服务器地址，可以指定多个。
- `-transport`
  指定传输协议类型。可以是 `TTHeader`、`Framed` 或 `TTHeaderFramed`，如不指定则为默认的 `Buffered`。
- `-biz-error`
  启用客户端接收 server 返回的业务错误。
- `-meta`
  指定传递给 server 的单跳透传元信息。可以指定多个，格式为 `key=value`。
- `-meta-persistent`
  指定传递给 server 的持续透传元信息。可以指定多个，格式为 `key=value`。
- `-meta-backward`
  启用从服务器接收反向透传元信息。
- `-q`
  只输出 Json 响应，不输出其他提示信息。
- `-verbose` 或 `-v`
  启用详细模式，提供更详细的输出信息。

### 详细描述

#### IDL 类型

使用 `-type` 或 `-t` 标志指定 IDL 类型。支持的类型有 `thrift` 和 `protobuf`，默认为 `thrift`。

```python
kitexcall -t thrift
```

#### IDL 路径

使用 `-idl-path` 或 `-p` 标志指定 IDL 文件的路径。

```python
kitexcall -idl-path /path/to/idl/file.thrift
```

#### 要调用的方法（必选）

使用 `-method` 或 `-m` 标志指定方法名。格式可以是 `IDLServiceName/MethodName` 或仅为 `MethodName`。当服务器端开启了 MultiService 模式时，必须指定 `IDLServiceName`，同时指定传输协议为 `TTHeader` 或 `TTHeaderFramed`。

```python
kitexcall -m GenericService/ExampleMethod
kitexcall -m ExampleMethod
```

#### 请求数据

使用 `-data` 或 `-d` 标志指定要发送的数据，数据应为 JSON 格式的字符串。或者，使用 `-file` 或 `-f` 标志指定包含数据的 JSON 文件。

假设我们要发送的数据为 `{"message": "hello"}`，我们可以这样指定数据：

```bash
kitexcall -m ExampleMethod -d '{"message": "hello"}'
```

或者，我们可以将数据保存在 JSON 文件中，比如 `input.json`，然后指定文件路径：

```bash
kitexcall -m ExampleMethod -f input.json
```

#### 服务器地址

使用 `-endpoint` 或 `-e` 标志指定一个或多个 server 地址。

假设我们有两个 server 地址 `127.0.0.1:9919` 和 `127.0.0.1:9920`，我们可以这样指定 server 地址：

```bash
kitexcall -m ExampleMethod -e 127.0.0.1:9919 -e 127.0.0.1:9920
```

#### 元信息

- 单跳透传（WithValue）：使用 `-meta` 标志指定，可以指定多个，格式为 `key=value`。
- 持续透传（WithPersistentValue）：使用 `-meta-persistent` 标志指定。可以指定多个，格式为 `key=value`。
- 反向透传元信息（Backward）：启用 `-meta-backward` 选项支持接收 server 返回的反向透传元信息（Backward）。

假设想要传递单跳透传元信息 `temp=temp-value` 和持续透传元信息 `logid=12345` 给 server，并接收来自反向透传元信息，可以这样指定：

```bash
kitexcall -m ExampleMethod -meta temp=temp-value -meta-persistent logid=12345 -meta-backward
```

#### 业务异常

如果 server 返回业务异常，可以通过 `-biz-error` 标志启用客户端接收业务自定义的异常错误码、错误信息及附加信息。

假设服务器返回业务错误状态码 `404` 和消息 `not found`，我们可以这样启用业务错误处理：

```bash
kitexcall -m ExampleMethod -biz-error
```

#### 启用详细模式

使用 `-verbose` 或 `-v` 标志启用详细模式，以提供更详细的输出信息。

### 流式调用支持

Kitexcall 支持 gRPC 流式 RPC 调用。在使用流式模式时，传输协议会自动设置为 gRPC。

#### 流式调用命令行选项

- `--streaming`：启用流式模式。流式类型会根据 IDL 文件中的方法定义自动判断。

#### 流式调用输入文件

对于需要发送多个消息的客户端流式和双向流式调用，您可以使用：
- 单个 JSON 文件（.json）用于发送单个消息
- JSONL 文件（.jsonl）用于发送多个消息，每行一个 JSON 对象

#### 示例

**客户端流式调用示例：**

```bash
cat > messages.jsonl << EOF
{"Msg": "hello 1"}
{"Msg": "hello 2"}
{"Msg": "hello 3"}
EOF
kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming -f messages.jsonl
```

**服务端流式调用示例：**

```bash
kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming -d '{"Msg": "hello"}'
```

**双向流式调用示例：**

```bash
kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming -f messages.jsonl
```

### 交互式模式

当未指定输入文件（`-f`）或数据（`-d`）时，kitexcall 会自动进入交互式模式。在交互式模式下，您可以：

1. 实时输入请求数据
2. 查看服务器响应
3. 继续输入新的请求数据
4. 使用 `Ctrl+D` 结束输入（不再发送新请求，但如果服务端继续推送，仍会接收流式响应）
5. 使用 `Ctrl+C` 终止流式会话（强制关闭连接）

示例：

```bash
kitexcall --idl-path echo.thrift --method TestService/EchoClient --endpoint 127.0.0.1:8888 --streaming

> {"Msg": "hello"}
[Status]: Success
{
    "Msg": "hello"
}
```

交互式模式特别适用于：
- 需要多次发送不同请求数据的场景
- 调试和测试服务接口
- 实时查看服务响应
