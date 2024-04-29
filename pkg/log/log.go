package log

import (
	"fmt"
	"os"
)

type Outputter interface {
	Success(data interface{}) error
	Fail(data interface{}) error
	Info(data interface{})
}

var Verbose bool

type Console struct{}

func (c *Console) Success(data interface{}) error {
	fmt.Println("\033[32mCall Status: Success\033[0m")
	fmt.Println(data)
	return nil
}

func (c *Console) Fail(data interface{}) error {

	fmt.Println("\033[31mCall Status: Failed\033[0m")
	fmt.Fprintln(os.Stderr, data)
	return nil
}

func (c *Console) Info(data interface{}) {
	if Verbose {
		fmt.Printf("[INFO]:%v\n", data)
	}
}

var console = &Console{}

func Info(data interface{}) {
	console.Info(data)
}

func Success(data interface{}) {
	console.Success(data)
}

func Fail(data interface{}) {
	console.Fail(data)
}

// For file output
type File struct{}

func (f *File) Success(data interface{}) error {
	return nil
}

func (f *File) Fail(data interface{}) error {
	return nil
}

func (f *File) Info(data interface{}) {

}
