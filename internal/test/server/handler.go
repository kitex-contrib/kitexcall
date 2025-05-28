package server

import (
	"fmt"
	"io"
	"time"

	streaming "github.com/kitex-contrib/kitexcall/internal/test/server/kitex_gen/streaming"
)

// StreamingServiceImpl implements the last service interface defined in the IDL.
type StreamingServiceImpl struct{}

// BidirectionalStream implements bidirectional streaming.
func (s *StreamingServiceImpl) BidirectionalStream(stream streaming.StreamingService_BidirectionalStreamServer) error {
	for {
		// Receive message from client
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Client closed the stream
				return nil
			}
			return err
		}

		// Process the message
		fmt.Printf("Received message: %s\n", msg.Msg)

		// Send response back to client
		resp := &streaming.StreamingResponse{
			Msg: &streaming.Message{
				Msg: fmt.Sprintf("Server received: %s", msg.Msg),
			},
			BaseResp: &streaming.BaseResp{
				StatusCode:    0,
				StatusMessage: "success",
			},
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}
}

// ServerStream implements server streaming.
func (s *StreamingServiceImpl) ServerStream(req *streaming.Message, stream streaming.StreamingService_ServerStreamServer) error {
	fmt.Printf("Received initial message: %s\n", req.Msg)

	// Send multiple responses
	for i := 0; i < 5; i++ {
		resp := &streaming.StreamingResponse{
			Msg: &streaming.Message{
				Msg: fmt.Sprintf("Server streaming response %d for: %s", i+1, req.Msg),
			},
			BaseResp: &streaming.BaseResp{
				StatusCode:    0,
				StatusMessage: "success",
			},
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// ClientStream implements client streaming.
func (s *StreamingServiceImpl) ClientStream(stream streaming.StreamingService_ClientStreamServer) error {
	var messageCount int
	var lastMessage string

	// Receive messages from client
	for {
		msg, err := stream.Recv()
		if err != nil {
			// Client has finished sending messages
			break
		}

		messageCount++
		lastMessage = msg.Msg
		fmt.Printf("Received message %d: %s\n", messageCount, msg.Msg)
	}

	// Send final response
	resp := &streaming.StreamingResponse{
		Msg: &streaming.Message{
			Msg: fmt.Sprintf("Processed %d messages, last message was: %s", messageCount, lastMessage),
		},
		BaseResp: &streaming.BaseResp{
			StatusCode:    0,
			StatusMessage: "success",
		},
	}

	return stream.SendAndClose(resp)
}
