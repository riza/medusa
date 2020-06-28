package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants"
	"github.com/valyala/fasthttp"
	"gopkg.in/gookit/color.v1"
)

const (
	medusaUA = "medusa/0.2.1"
	version  = "0.2.1"
)

const (
	urlTypeSingle = iota
	urlTypeMultiple

	defaultSchema = "http"
	schemaSlicer  = "://"

	sameBodyLengthCountAlarm = 3
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
	v         = flag.Bool("v", false, "")
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

	sameBodyWarningHost, currentBody atomic.Value
	sameBodyLengthCount              int32
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
-r                    Enable recursive fuzzing
-w                    Directory wordlist (line by line)
-v                    Verbose mode, show logs
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
[*] Max concurrent connection: %d
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
		*conc,
	)

	wg = &sync.WaitGroup{}

	client = &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Duration(*t)*time.Second)
		},
		TLSConfig: &tls.Config{InsecureSkipVerify: *insecure},
	}

	start := time.Now()

	p, _ = ants.NewPoolWithFunc(*conc, func(u interface{}) {
		check(u)
		wg.Done()
	})

	defer p.Release()

	for _, u := range urls {
		invokeAndGeneratePath(u)
	}

	wg.Wait()

	took := time.Since(start)
	logInfo(fmt.Sprintf("Total time:\t%s", took))

}

func check(i interface{}) {
	url := i.(string)

	host := getHost(url)

	//skip samebody
	if sameBodyWarningHost.Load() != nil && host == sameBodyWarningHost.Load().(string) {
		return
	}

	sc, body, err := client.Get(nil, url)
	if err != nil {
		if err != fasthttp.ErrDialTimeout {
			logError(err.Error())
		}
	}

	statusCode := strconv.Itoa(sc)
	if !checkStatusCode(nCodes, statusCode) && checkStatusCode(pCodes, statusCode) {
		if currentBody.Load() != nil && currentBody.Load().(int) == len(body) {
			if sameBodyLengthCount >= sameBodyLengthCountAlarm {
				if sameBodyWarningHost.Load() == nil {
					sameBodyWarningHost.Store(host)
					logInfo("Same body detected, skipping host: " + host)
					return
				}

				if host != sameBodyWarningHost.Load().(string) {
					sameBodyWarningHost.Store(host)
				}
				return
			}
			atomic.AddInt32(&sameBodyLengthCount, 1)
		} else {

			if sameBodyLengthCount != 0 {
				sameBodyLengthCount = 0
			}

			if *recursive {
				go invokeAndGeneratePath(url)
			}

			currentBody.Store(len(body))

			logFound(url, statusCode, strconv.Itoa(len(body)))
		}
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

func getHost(u string) string {
	pu, err := url.Parse(u)

	if err != nil {
		return ""
	}

	return pu.Host
}

func checkStatusCode(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func generateURL(u string) string {
	if !strings.Contains(u, "http") || !strings.Contains(u, "https") {
		if len(*s) <= 0 {
			u = defaultSchema + schemaSlicer + u
		}
	}

	if len(*s) > 0 {
		if strings.Contains(u, "http") {
			strings.ReplaceAll(u, "http", *s)
		} else if strings.Contains(u, "https") {
			strings.ReplaceAll(u, "https", *s)
		} else {
			u = *s + schemaSlicer + u
		}
	}

	if u[len(u)-1:] != "/" {
		u = u + "/"
	}

	return u
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
	color.Green.Print("[" + status + "] ")
	fmt.Println(url + " - " + length)
}

func logInfo(msg string) {
	color.Yellow.Print("[+] ")
	fmt.Println(msg)
}

func logError(msg string) {
	color.Red.Print("[ERR] ")
	fmt.Println(msg)
}

func usageAndExit(msg string) {
	if msg != "" {
		logError(msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
