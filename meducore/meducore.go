package medusa

import (
	"io/ioutil"
	"log"
	"medusa/meduhttp"
	"strings"
	"sync"
	"time"

	"github.com/paulbellamy/ratecounter"
)

type Found struct {
	sync.RWMutex
	List []Page
}

type Path struct {
	List []string
}

type URL struct {
	List []string
}

type Page struct {
	StatusCode int
	URL        string
	Body       []byte
	OutputOK   bool
}

type Options struct {
	Wg         *sync.WaitGroup
	Timeout    time.Duration
	HTTPClient *meduhttp.HTTPClient
	Boundary   chan struct{}
	RetryLimit int
}

type RetryPath struct {
	sync.RWMutex
	List map[string]map[string]int
}

type Medusa struct {
	*Options
	*RetryPath
	Counter *ratecounter.RateCounter
}

func New(options *Options) *Medusa {

	retry := &RetryPath{
		List: make(map[string]map[string]int),
	}

	return &Medusa{
		options,
		retry,
		ratecounter.NewRateCounter(1 * time.Second),
	}
}

func (m *Medusa) Check(host, dir string, response chan *meduhttp.Response) {

	url := host + "/" + dir

	m.Boundary <- struct{}{}

	time.Sleep(time.Second)

	resp, err := m.HTTPClient.Get(url)

	if err != nil {
		if strings.Contains(err.Error(), "too many open files") || strings.Contains(err.Error(), "connection reset by peer") || strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "no such host") {

			m.Lock()
			retryPathHost, ok := m.RetryPath.List[url]

			if !ok {
				m.RetryPath.List[url] = make(map[string]int)
			}

			_, ok = retryPathHost[dir]

			if !ok {
				m.RetryPath.List[url][dir] = 1
			} else {

				if m.RetryLimit > m.RetryPath.List[url][dir] {
					log.Fatal("Retry limit exceed for -> ", url)
				}

				m.RetryPath.List[url][dir]++
			}
			m.Unlock()

			time.Sleep(5 * time.Second)
			m.Check(host, dir, response)
			return
		} else {
			log.Fatal(err)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	m.Counter.Incr(1)

	response <- &meduhttp.Response{
		StatusCode: resp.StatusCode,
		URL:        url,
		Body:       body,
	}

	resp.Close = true

	defer m.Wg.Done()
	<-m.Boundary

}
