package common

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// ParseInputAndOutput 解析输入的文件
func ParseInputAndOutput(input, output string) (inputReader, outWriter *os.File, err error) {
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

// ParseInput 解析输入的文件
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

// PrintErrorMsgAndExit 打印信息并退出
func PrintErrorMsgAndExit(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s Error:%v\n", msg, err)
	os.Exit(1)
}

// LF `\n`
const LF = '\n'

// ProcessLineFunc 行处理函数
type ProcessLineFunc func(data string, lineNum int, readErr error) (stop bool)

// ProcessLines 按行从rd中读取数据,交由processFunc进行处理
func ProcessLines(rd io.Reader, processFunc ProcessLineFunc) {
	scanner := bufio.NewReaderSize(rd, 4*1024)
	var readErr error
	var lineNum = 0
	var data string
	for readErr == nil {
		data, readErr = scanner.ReadString(LF)
		lineNum++
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

// ProcessFileLines 按行处理文件
func ProcessFileLines(file string, lineFunc ProcessLineFunc) {
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	ProcessLines(f, lineFunc)
}

// WaitStop 等待退出信号
func WaitStop() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-c
	return s
}

// CreateDirIfAbsent 当目录不存在时创建
func CreateDirIfAbsent(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("can't create dir:%s,err:%s", dir, err)
			}
		}
	} else if !info.IsDir() {
		return fmt.Errorf("not a dir `%s`", dir)
	}
	return nil
}
