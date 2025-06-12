/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

namespace go streaming

struct Message {
    1: string Msg
}

struct BaseResp {
    1: i32 StatusCode
    2: string StatusMessage
}

struct StreamingResponse {
    1: Message Msg
    2: BaseResp BaseResp
}

service StreamingService {
    // Bidirectional streaming
    StreamingResponse BidirectionalStream(1: Message req) (streaming.mode="bidirectional"),
    
    // Server streaming
    StreamingResponse ServerStream(1: Message req) (streaming.mode="server"),
    
    // Client streaming
    StreamingResponse ClientStream(1: Message req) (streaming.mode="client"),
} 