package scraper

import (
	"errors"
	"runtime"
	"time"

	"github.com/olebedev/emitter"
)

// StreamRunner runs a stream of urls through a scraper
type StreamRunner struct {
	scraper  Scraper
	inputURL chan string
	emitter  *emitter.Emitter
}

func (s *StreamRunner) worker(id int, urls <-chan string) {
	var done chan struct{}
	for url := range urls {
		res, err := s.scraper.ScrapeURL(url)
		if err == nil {
			rows, err := s.scraper.GetRows(res)
			if err == nil {
				done = s.emitter.Emit(url+":result", rows)
				select {
				case <-done:
					// so the sending is done
				case <-time.After(5):
					// time is out, let's discard emitting
					close(done)
				}
				continue
			}
		}
		done = s.emitter.Emit(url+":error", err)
		select {
		case <-done:
			// so the sending is done
		case <-time.After(5):
			// time is out, let's discard emitting
			close(done)
		}
	}
}

func (s *StreamRunner) pool() *StreamRunner {
	for w := 1; w <= runtime.NumCPU()-1; w++ {
		go s.worker(w, s.inputURL)
	}
	return s
}

//Add a url to the stream
func (s *StreamRunner) Add(url string) bool {
	s.inputURL <- url
	return true
}

//GetResult of a url
func (s *StreamRunner) GetResult(url string) <-chan emitter.Event {
	return s.emitter.Once(url + ":result")
}

//GetError of a url
func (s *StreamRunner) GetError(url string) <-chan emitter.Event {
	return s.emitter.Once(url + ":error")
}

// OneResult waits and returns a result or an error if a scrape request failed
func (s *StreamRunner) OneResult(path string) (record map[string]interface{}, err error) {
OUTER:
	for {
		select {
		case ev := <-s.GetError(path):
			err = ev.Args[0].(error)
			break OUTER
		case ev := <-s.GetResult(path):
			vi := ev.Args[0].([]interface{})
			if len(vi) == 1 {
				record = vi[0].(map[string]interface{})
			}
			err = errors.New("domain not found")
			break OUTER
		}
	}
	return
}

//Close runner
func (s *StreamRunner) Close() {
	close(s.inputURL)
}

// NewStreamRunner creates a pointer to a new stream runner
func NewStreamRunner(scraper Scraper) *StreamRunner {
	s := &StreamRunner{
		scraper:  scraper,
		inputURL: make(chan string),
		emitter:  &emitter.Emitter{},
	}
	return s.pool()
}
