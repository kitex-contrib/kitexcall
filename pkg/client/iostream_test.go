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

package client

import (
	"bytes"
	"io"
	"testing"
)

func TestIoStream_Recv(t *testing.T) {
	t.Run("one valid JSON request", func(t *testing.T) {
		oriReq := "{\"hello\": \"world\"}\n"
		mockReader := bytes.NewBufferString(oriReq)
		st := newIoStream(mockReader, nil)
		req, err := st.Recv()
		if err != nil {
			t.Errorf("Recv() error = %v", err)
		}
		if req != "{\"hello\": \"world\"}" {
			t.Errorf("Recv() got = %v, want %v", req, "{\"hello\": \"world\"}\n")
		}
		_, err = st.Recv()
		if err != io.EOF {
			t.Errorf("Recv() error = %v, want %v", err, io.EOF)
		}
	})

	t.Run("multiple valid JSON requests", func(t *testing.T) {
		oriReq := "{\"hello\": \"world\"}\n{\"hello\": \"world\"}\n{\"hello\": \"world\"}\n"
		mockReader := bytes.NewBufferString(oriReq)
		st := newIoStream(mockReader, nil)
		for i := 0; i < 3; i++ {
			req, err := st.Recv()
			if err != nil {
				t.Errorf("Recv() error = %v", err)
			}
			if req != "{\"hello\": \"world\"}" {
				t.Errorf("Recv() got = %v, want %v", req, "{\"hello\": \"world\"}\n")
			}
		}
		_, err := st.Recv()
		if err != io.EOF {
			t.Errorf("Recv() error = %v, want %v", err, io.EOF)
		}
	})
}
