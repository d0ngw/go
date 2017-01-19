package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func ParseInputAndOutPut(input, output string) (inputReader, outWriter *os.File, err error) {
	if input == "-" {
		inputReader = os.Stdin
		fmt.Fprintln(os.Stderr, "Read data from stdin")
	} else if input != "" {
		fmt.Fprintln(os.Stderr, "Read data from "+input)
		fileInput, err := os.Open(input)
		if err == nil {
			inputReader = fileInput
		}
	}

	if output == "" {
		fmt.Fprintln(os.Stderr, "Write data to stdout")
		outWriter = os.Stdout
	} else {
		fmt.Fprintln(os.Stderr, "Write data to "+output)
		fileOut, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err == nil {
			outWriter = fileOut
		}
	}

	var errorMsg string
	if inputReader == nil {
		errorMsg = "Invalid input:" + input
		if outWriter != nil {
			defer outWriter.Close()
		}
	}

	if outWriter == nil {
		if len(errorMsg) > 0 {
			errorMsg += "Invalid output:" + output
		}
		if inputReader != nil {
			defer inputReader.Close()
		}
	}

	if errorMsg != "" {
		return nil, nil, errors.New(errorMsg)
	}

	return
}

func ParseInput(input string) (inputReader *os.File, err error) {
	if input == "-" {
		inputReader = os.Stdin
		fmt.Fprintln(os.Stderr, "Read data from stdin")
	} else if input != "" {
		fmt.Fprintln(os.Stderr, "Read data from "+input)
		fileInput, err := os.Open(input)
		if err == nil {
			inputReader = fileInput
		}
	}

	var errorMsg string
	if inputReader == nil {
		errorMsg = "Invalid input:" + input
	}

	if errorMsg != "" {
		return nil, errors.New(errorMsg)
	}

	return
}

func PrintErrorMsgAndExit(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s Error:%v\n", msg, err)
	os.Exit(1)
}

const LF = '\n'

type ProcessLineFunc func(data string, lineNum int, readErr error) (stop bool)

// 按行从rd中读取数据,交由processFunc进行处理
func ProcessLines(rd io.Reader, processFunc ProcessLineFunc) {
	scanner := bufio.NewReaderSize(rd, 4*1024)
	var readErr error
	var lineNum = 0
	var data string
	for readErr == nil {
		data, readErr = scanner.ReadString(LF)
		lineNum++

		data = strings.TrimSpace(data)
		if readErr != nil && readErr != io.EOF {
			processFunc(data, lineNum, readErr)
			break
		} else if readErr != nil && readErr == io.EOF {
			if len(data) > 0 {
				processFunc(data, lineNum, nil)
			}
			break
		} else {
			if len(data) > 0 {
				if processFunc(data, lineNum, nil) {
					break
				}
			}
		}
	}
}

func WaitStop() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-c
	return s
}
