package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type ResponseDetails struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
	Error      string `json:"error,omitempty"`
}

func main() {
	quiet := false
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-q" {
			quiet = true
			args = append(args[:i], args[i+1:]...)
			i-- // adjust index after slice modification
		}
	}

	var scanner *bufio.Scanner
	if len(args) > 0 {
		file, err := os.Open(args[0])
		if err != nil {
			if !quiet {
				fmt.Println("Error opening file:", err)
			}
			return
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	// Custom HTTP transport
	tr := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90,
		MaxIdleConnsPerHost: 10,
	}
	client := &http.Client{Transport: tr}

	var wg sync.WaitGroup
	var outputLock sync.Mutex
	var output bytes.Buffer

	for scanner.Scan() {
		url := scanner.Text()
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := client.Get(url)
			details := ResponseDetails{URL: url}
			if err != nil {
				details.Error = err.Error()
			} else {
				defer resp.Body.Close()
				details.StatusCode = resp.StatusCode
				details.StatusMsg = resp.Status
			}
			if !quiet {
				jsonOutput, _ := json.Marshal(details)
				outputLock.Lock()
				output.Write(jsonOutput)
				output.WriteByte('\n')
				outputLock.Unlock()
			}
		}(url)
	}

	wg.Wait()
	if !quiet {
		fmt.Print(output.String())
	}
	if scanner.Err() != nil && !quiet {
		fmt.Println("Error reading input:", scanner.Err())
	}
}
