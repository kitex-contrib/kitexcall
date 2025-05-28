package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/kitex-contrib/kitexcall/pkg/log"
)

type outputFormatter struct{}

func (f *outputFormatter) format(output any) (string, error) {
	if outputStr, ok := output.(string); ok {
		return log.FormatJson(outputStr)
	}

	return "", fmt.Errorf("output of %T is not supported, now only support string", output)
}

// ioStream is responsible for interacting with standard input/output or file input/output.
// it provides generic_client with streaming interface.
type ioStream struct {
	// deal with Recv input from
	decoder *json.Decoder
	err     error

	// deal with Send output to
	out       io.Writer
	formatter *outputFormatter

	// signal channel for handling Ctrl+C
	sigChan chan os.Signal
}

func newIoStream(in io.Reader, out io.Writer) *ioStream {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	return &ioStream{
		decoder:   json.NewDecoder(in),
		out:       out,
		formatter: &outputFormatter{},
		sigChan:   sigChan,
	}
}

// WithInterruptHandler creates a new context that will be cancelled when Ctrl+C is pressed
func (st *ioStream) WithInterruptHandler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-st.sigChan:
			log.Info("Stream cancelled by user (Ctrl+C)")
			cancel()
		case <-ctx.Done():
			// Context was cancelled by other means
		}
	}()
	return ctx
}

func (st *ioStream) Recv() (string, error) {
	if st.err != nil {
		return "", st.err
	}

	for {
		var jsonMsg json.RawMessage
		if err := st.decoder.Decode(&jsonMsg); err != nil {
			st.err = err
			return "", err
		}
		return string(jsonMsg), nil
	}
}

func (st *ioStream) Send(res any) error {
	resStr, err := st.formatter.format(res)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(st.out, "%s\n", resStr)
	return err
}
