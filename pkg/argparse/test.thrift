namespace go test

struct Request {
    1: required string message
}

struct Response {
    1: required string message
}

service TestService {
    Response Method(1: Request req) (streaming.mode="bidirectional")
    Response ClientMethod(1: Request req) (streaming.mode="client")
    Response ServerMethod(1: Request req) (streaming.mode="server")
} 