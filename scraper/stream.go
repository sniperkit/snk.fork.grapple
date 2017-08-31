package scraper

import (
	"runtime"

	"github.com/olebedev/emitter"
)

// StreamRunner runs a stream of urls through a scraper
type StreamRunner struct {
	scraper  Scraper
	inputURL chan string
	emitter  *emitter.Emitter
}

func (s *StreamRunner) worker(id int, urls <-chan string) {
	for url := range urls {
		res, err := s.scraper.ScrapeURL(url)
		if err == nil {
			rows, err := s.scraper.GetRows(res)
			if err == nil {
				<-s.emitter.Emit(url+":result", rows)
				continue
			}
		}
		<-s.emitter.Emit(url+":error", err)
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
	return s.emitter.On(url + ":result")
}

//GetError of a url
func (s *StreamRunner) GetError(url string) <-chan emitter.Event {
	return s.emitter.On(url + ":error")
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