// Copyright 2024 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

namespace go kitex.test.server

include "common.thrift"

// Request data for Example2Service
struct Example2Req {
    1: required string Msg,
    2: required common.UserData User,
}

// Response for Example2Service
struct Example2Resp {
    1: required string Msg,
    2: required common.CommonResponse BaseResp,
}

// Service definition that uses imported common types
service Example2Service {
    Example2Resp Example2Method(1: Example2Req req),
}
