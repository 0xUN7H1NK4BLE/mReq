package main

import (
	"flag"
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
	help := false

	flag.BoolVar(&quiet, "q", false, "Run in quiet mode (suppress output)")
	flag.BoolVar(&help, "h", false, "Display help message")
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	args := flag.Args()

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
	
	// Number of worker goroutines
	numWorkers := 10

	// Channel to send URLs to workers
	urlChan := make(chan string, numWorkers*10) // Buffer size is multiple of numWorkers to minimize blocking

	var wg sync.WaitGroup
	var outputLock sync.Mutex
	var output bytes.Buffer

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlChan {
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
			}
		}()
	}

	// Feed URLs to the worker pool
	for scanner.Scan() {
		url := scanner.Text()
		urlChan <- url
	}

	// Close the URL channel to signal workers that no more URLs are coming
	close(urlChan)

	// Wait for all workers to finish
	wg.Wait()
	if !quiet {
		fmt.Print(output.String())
	}
	if scanner.Err() != nil && !quiet {
		fmt.Println("Error reading input:", scanner.Err())
	}
	// Exit the program
	os.Exit(0)
}
