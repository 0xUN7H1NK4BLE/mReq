package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/fatih/color" // Make sure to import the color package
)

type ResponseDetails struct {
	URL        string
	StatusCode int
	StatusMsg  string
	Error      string
}

func main() {
	quiet := false
	args := os.Args[1:]
	remainingArgs := []string{}

	// Manually parse arguments for the '-q' quiet flag
	for _, arg := range args {
		if arg == "-q" {
			quiet = true
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	var scanner *bufio.Scanner
	if len(remainingArgs) > 0 {
		file, err := os.Open(remainingArgs[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			return
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	tr := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * 1e9,
		MaxIdleConnsPerHost: 10,
	}
	client := &http.Client{Transport: tr}

	numWorkers := 10
	urlChan := make(chan string, numWorkers*10)

	var wg sync.WaitGroup
	var output sync.Mutex
	var outputBuffer string

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
					output.Lock()
					var outputLine string
					if details.Error != "" {
						outputLine = color.RedString(fmt.Sprintf("%s [ERROR]", details.URL))
					} else {
						// Color output based on status code
						switch {
						case details.StatusCode >= 200 && details.StatusCode < 300:
							outputLine = color.GreenString(fmt.Sprintf("%s [%d]", details.URL, details.StatusCode))
						case details.StatusCode >= 300 && details.StatusCode < 400:
							outputLine = color.CyanString(fmt.Sprintf("%s [%d]", details.URL, details.StatusCode))
						case details.StatusCode >= 400 && details.StatusCode < 500:
							outputLine = color.YellowString(fmt.Sprintf("%s [%d]", details.URL, details.StatusCode))
						case details.StatusCode >= 500:
							outputLine = color.RedString(fmt.Sprintf("%s [%d]", details.URL, details.StatusCode))
						default:
							outputLine = fmt.Sprintf("%s [%d]", details.URL, details.StatusCode)
						}
					}
					outputBuffer += outputLine + "\n"
					output.Unlock()
				}
			}
		}()
	}

	for scanner.Scan() {
		urlChan <- scanner.Text()
	}

	close(urlChan)
	wg.Wait()

	if !quiet {
		fmt.Print(outputBuffer)
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", scanner.Err())
	}
}
