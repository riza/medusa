package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	medusa "medusa/meducore"
	meduhttp "medusa/meduhttp"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gosuri/uilive"
)

var (

	//flags
	flagInputHostListPath, flagInputDirListPath,
	flagInputHost, flagInputDir, flagCollectCode, flagOutputFileName string

	flagMaxProcs, flagBoundary, flagRetryLimitPerHost, flagDepth int

	flagTimeout time.Duration

	//init vals
	paths = medusa.Path{}
	urls  = medusa.URL{}
	found = medusa.Found{}

	collectCodes = make(map[string]byte)

	green  = color.New(color.Bold, color.FgHiGreen).SprintFunc()
	yellow = color.New(color.Bold, color.FgHiYellow).SprintFunc()
	cyan   = color.New(color.Bold, color.FgHiCyan).SprintFunc()
	red    = color.New(color.Bold, color.FgHiRed).SprintFunc()
	white  = color.New(color.Bold, color.FgHiWhite).SprintFunc()

	version = "0x00"

	rspLine string
	err     error
)

func init() {
	//singular
	flag.StringVar(&flagInputHost, "h", "", "Singular host")
	flag.StringVar(&flagInputDir, "d", "", "Singular dir")

	//multiple
	flag.StringVar(&flagInputHostListPath, "H", "", "Multiple host from file")
	flag.StringVar(&flagInputDirListPath, "D", "", "Multiple dir from file")

	//performance
	flag.IntVar(&flagMaxProcs, "cpu", 0, "CPU Procs")
	flag.IntVar(&flagBoundary, "boundary", 1024, "Concurrent boundary limit")
	flag.IntVar(&flagRetryLimitPerHost, "retry", 3, "Retry limit per host")
	flag.IntVar(&flagDepth, "depth", 0, "Depth level")

	flag.StringVar(&flagCollectCode, "cc", "200", "Collect status code (--c 403,404)")
	flag.StringVar(&flagOutputFileName, "o", "output.txt", "Output filename")

	flag.DurationVar(&flagTimeout, "timeout", 500, "HTTP response timeout")

}

func main() {

	flag.Parse()

	if len(flagCollectCode) >= 1 {
		splitStatusCodes := strings.Split(flagCollectCode, ",")

		for _, code := range splitStatusCodes {
			_, ok := collectCodes[code]

			if !ok {
				collectCodes[code] = 0x00
			}
		}
	}

	if flagMaxProcs >= 1 {
		runtime.GOMAXPROCS(flagMaxProcs)
	} else {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	if len(flagInputDirListPath) > 2 {
		paths.List, err = readLines(flagInputDirListPath)

		if err != nil {
			log.Fatal(err)
		}

		goto INITHOSTFILE
	}

	if len(flagInputDir) > 2 {
		paths.List = append(paths.List, flagInputDir)

		goto INITHOSTFILE
	}

INITHOSTFILE:

	if len(flagInputHostListPath) > 2 {
		urls.List, err = readLines(flagInputHostListPath)

		if err != nil {
			log.Fatal(err)
		}

		goto MAIN
	}

	if len(flagInputHost) > 2 {
		urls.List = append(urls.List, flagInputHost)

		goto MAIN
	}

MAIN:

	start := time.Now()

	httpClient := meduhttp.New(&meduhttp.Options{
		Timeout: flagTimeout * time.Millisecond,
	})

	app := medusa.New(&medusa.Options{
		Wg:         &sync.WaitGroup{},
		HTTPClient: httpClient,
		Boundary:   make(chan struct{}, flagBoundary),
		RetryLimit: flagRetryLimitPerHost,
	})

	defer close(app.Boundary)

	response := make(chan *meduhttp.Response)

	for _, url := range urls.List {
		for _, dir := range paths.List {
			app.Wg.Add(1)
			go app.Check(url, dir, response)
		}
	}

	outputFile, err := os.Create(flagOutputFileName)

	if err != nil {
		log.Fatal(err)
	}

	defer outputFile.Close()

	fileWriter := bufio.NewWriter(outputFile)

	go writeFileAsync(fileWriter, &found)

	writer := uilive.New()
	writer.Start()

	// UI

	fmt.Printf("%s v1.0 - %s \n----\n", cyan("Medusa Alpha"), version)
	fmt.Printf("%s: %d \n", green("Total host"), len(urls.List))
	fmt.Printf("%s: %d \n", green("Total dir"), len(paths.List))

	go func() {
		for {
			rsp := <-response

			switch rsp = rsp; {
			case rsp.StatusCode == 200:
				rspLine = fmt.Sprintf(RESP_OUTPUT_NEWLINE, green("200"), rsp.URL)
				break
			case rsp.StatusCode >= 400:
				rspLine = fmt.Sprintf(RESP_OUTPUT_NEWLINE, yellow("404"), rsp.URL)
				break
			case rsp.StatusCode >= 500:
				rspLine = fmt.Sprintf(RESP_OUTPUT_NEWLINE, red("500"), rsp.URL)
				break
			default:
				rspLine = fmt.Sprintf(RESP_OUTPUT_NEWLINE, white(string(rsp.StatusCode)), rsp.URL)
				break
			}

			_, collect := collectCodes[strconv.Itoa(rsp.StatusCode)]

			if collect {
				found.Lock()
				found.List = append(found.List, medusa.Page{
					StatusCode: rsp.StatusCode,
					URL:        rsp.URL,
					Body:       rsp.Body,
				})
				found.Unlock()
			}

			time.Sleep(3 * time.Millisecond)
			fmt.Fprintf(writer, "%s: %dreq/s\n\n", green("Current freq"), app.Counter.Rate())
			fmt.Fprintf(writer, "%s", rspLine)
		}
	}()

	app.Wg.Wait()

	fmt.Fprintln(writer, fmt.Sprintf(RESP_FINISH_NEWLINE, green("FINISH"), time.Since(start).Seconds()))
	writer.Stop()
}
