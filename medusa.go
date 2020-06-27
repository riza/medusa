package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants"
	"github.com/valyala/fasthttp"
)

const (
	medusaUA = "medusa/0.2.0"
	version  = "0.2.0"
)

const (
	urlTypeSingle = iota
	urlTypeMultiple

	defaultSchema = "http"
	schemaSlicer  = "://"
)

var (
	u  = flag.String("u", "", "")
	ul = flag.String("uL", "", "")

	ua = flag.String("ua", medusaUA, "")

	cp        = flag.String("cP", "200", "")
	cn        = flag.String("cN", "", "")
	e         = flag.String("e", "", "")
	s         = flag.String("s", "", "")
	t         = flag.Int("t", 10, "")
	insecure  = flag.Bool("x", false, "")
	recursive = flag.Bool("r", false, "")

	wl = flag.String("w", "", "")

	conc = flag.Int("conc", 100, "")

	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")

	client *fasthttp.Client
	p      *ants.PoolWithFunc
	wg     *sync.WaitGroup

	pCodes = []string{}
	nCodes = []string{}

	wordlist = []string{}
	urls     = []string{}
)

var usage = `Usage: medusa [options...]
Options:
-u                    Single URL  
-uL                   URL list file path (line by line)
-e                    Extension 
-s                    Force schema (uses default http if does not contains url)
-ua                   User-agent value (default %s)
-cP                   Postive status codes (seperate by comma)
-cN                   Negative status codes (seperate by comma)
-x                    Bypass SSL verification
-t                    HTTP response timeout (10s)
-r                    Enable recursive fuzzing *
-w                    Directory wordlist (line by line)
-conc                 Maximum concurrent requests
-cpus                 Number of used cpu cores.
(default for current machine is %d cores)
`

var header = `
medusa v%s by rizasabuncu
-----------------------------------------
[*] URL/List: %s
[*] CPU Cores: %d
[*] User Agent: %s
[*] Extension: %s
[*] Positive status codes: %s
[*] Negative status codes: %s
[*] Bypass SSL verification %t
[*] Recursive fuzz: %t
[*] Wordlist path: %s
[*] Wordlist length: %d
-----------------------------------------
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, medusaUA, runtime.NumCPU()))
	}

	flag.Parse()
	if flag.NFlag() < 1 {
		usageAndExit("")
	}

	if len(*u)+len(*ul) <= 0 {
		usageAndExit("-u or -uL cannot be empty")
	}

	if len(*wl) <= 0 {
		usageAndExit("-w cannot be empty")
	}

	runtime.GOMAXPROCS(*cpus)
	defer ants.Release()

	var (
		urlType = urlTypeSingle

		err     error
		scanURL = *u
	)

	pCodes = parseStatusCode(*cp)
	nCodes = parseStatusCode(*cn)

	if len(*ul) > len(*u) {
		urlType = urlTypeMultiple
		scanURL = *ul
	}

	switch urlType {
	case urlTypeSingle:
		urls = append(urls, scanURL)
		break
	case urlTypeMultiple:

		_, err = os.Stat(scanURL)
		if err != nil {
			usageAndExit("URL list file not exists")
		}

		urls, err = readLines(scanURL)
		if err != nil {
			usageAndExit("URL list load error: " + err.Error())
		}

		break
	}

	_, err = os.Stat(*wl)
	if err != nil {
		usageAndExit("Wordlist file not exists")
	}

	wordlist, err = readLines(*wl)
	if err != nil {
		usageAndExit("Wordlist load error: " + err.Error())
	}

	//header
	fmt.Printf(
		header, version,
		scanURL, *cpus,
		*ua, *e,
		*cp, *cn,
		*insecure,
		*recursive,
		*wl, len(wordlist),
	)

	wg = &sync.WaitGroup{}

	client = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Duration(*t)*time.Second)
		},
		TLSConfig: &tls.Config{InsecureSkipVerify: *insecure},
	}

	start := time.Now()

	p, _ = ants.NewPoolWithFunc(*conc, func(i interface{}) {
		check(i)
		wg.Done()
	})

	defer p.Release()

	for _, u := range urls {
		invokeAndGeneratePath(u)
	}

	wg.Wait()

	took := time.Since(start)

	fmt.Printf("Total time:\t%s", took)
	fmt.Printf("running goroutines: %d\n", p.Running())
}

func check(i interface{}) {
	url := i.(string)

	sc, body, err := client.Get(nil, url)
	if err != nil {
		logError(err.Error())
	}

	statusCode := strconv.Itoa(sc)

	if !checkStatusCode(nCodes, statusCode) && checkStatusCode(pCodes, statusCode) {
		if *recursive {
			go invokeAndGeneratePath(url)
		}

		logFound(url, statusCode, strconv.Itoa(len(body)))
	}
}

func invokeAndGeneratePath(u string) {
	u = generateURL(u)
	wg.Add(len(wordlist))

	for _, w := range wordlist {
		_ = p.Invoke(u + w + *e)
	}
}

func parseStatusCode(codes string) []string {
	return strings.Split(codes, ",")
}

func checkStatusCode(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func generateURL(url string) string {
	if !strings.Contains(url, "http") || !strings.Contains(url, "https") {
		if len(*s) <= 0 {
			url = defaultSchema + schemaSlicer + url
		}
	}

	if len(*s) > 0 {
		if strings.Contains(url, "http") {
			strings.ReplaceAll(url, "http", *s)
		} else if strings.Contains(url, "https") {
			strings.ReplaceAll(url, "https", *s)
		} else {
			url = *s + schemaSlicer + url
		}
	}

	if url[len(url)-1:] != "/" {
		url = url + "/"
	}

	return url
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func logFound(url, status, length string) {
	fmt.Fprintf(os.Stdin, "["+status+"] "+url+" - "+length+" \n")
}

func logInfo(msg string) {
	fmt.Fprintf(os.Stdin, "[+] "+msg+"\n")
}

func logError(msg string) {
	fmt.Fprintf(os.Stdin, "[X] "+msg+"\n")
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
