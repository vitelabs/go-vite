package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-config/reader"
	"github.com/micro/go-config/reader/json"
	"github.com/micro/go-config/source"
)

type config struct {
	exit chan bool
	opts Options

	sync.RWMutex
	// the current merged set
	set *source.ChangeSet
	// the current values
	vals reader.Values
	// all the sets
	sets []*source.ChangeSet
	// all the sources
	sources []source.Source

	idx      int
	watchers map[int]*watcher
}

type watcher struct {
	exit    chan bool
	path    []string
	value   reader.Value
	updates chan reader.Value
}

func newConfig(opts ...Option) Config {
	options := Options{
		Reader: json.NewReader(),
	}

	for _, o := range opts {
		o(&options)
	}

	c := &config{
		exit:     make(chan bool),
		opts:     options,
		watchers: make(map[int]*watcher),
		sources:  options.Source,
	}

	for i, s := range options.Source {
		go c.watch(i, s)
	}
	return c
}

func (c *config) watch(idx int, s source.Source) {
	c.Lock()
	c.sets = append(c.sets, &source.ChangeSet{Source: s.String()})
	c.Unlock()

	// watches a source for changes
	watch := func(idx int, s source.Watcher) error {
		for {
			// get changeset
			cs, err := s.Next()
			if err != nil {
				return err
			}

			c.Lock()

			// save
			c.sets[idx] = cs

			// merge sets
			set, err := c.opts.Reader.Merge(c.sets...)
			if err != nil {
				c.Unlock()
				return err
			}

			// set values
			c.vals, _ = c.opts.Reader.Values(set)
			c.set = set

			c.Unlock()

			// send watch updates
			c.update()
		}
	}

	for {
		// watch the source
		w, err := s.Watch()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		done := make(chan bool)

		// the stop watch func
		go func() {
			select {
			case <-done:
			case <-c.exit:
			}
			w.Stop()
		}()

		// block watch
		if err := watch(idx, w); err != nil {
			// do something better
			time.Sleep(time.Second)
		}

		// close done chan
		close(done)

		// if the config is closed exit
		select {
		case <-c.exit:
			return
		default:
		}
	}
}

func (c *config) loaded() bool {
	var loaded bool
	c.RLock()
	if c.vals != nil {
		loaded = true
	}
	c.RUnlock()
	return loaded
}

func (c *config) update() {
	var watchers []*watcher

	c.RLock()
	for _, w := range c.watchers {
		watchers = append(watchers, w)
	}
	c.RUnlock()

	for _, w := range watchers {
		select {
		case w.updates <- c.vals.Get(w.path...):
		default:
		}
	}
}

func (c *config) Map() map[string]interface{} {
	c.RLock()
	defer c.RUnlock()
	return c.vals.Map()
}

func (c *config) Scan(v interface{}) error {
	c.RLock()
	defer c.RUnlock()
	return c.vals.Scan(v)
}

// sync loads all the sources, calls the parser and updates the config
func (c *config) Sync() error {
	var sets []*source.ChangeSet

	c.Lock()

	// read the source
	var gerr []string

	for _, source := range c.sources {
		ch, err := source.Read()
		if err != nil {
			gerr = append(gerr, err.Error())
			continue
		}
		sets = append(sets, ch)
	}

	// merge sets
	set, err := c.opts.Reader.Merge(sets...)
	if err != nil {
		c.Unlock()
		return err
	}

	// set values
	vals, err := c.opts.Reader.Values(set)
	if err != nil {
		c.Unlock()
		return err
	}
	c.vals = vals
	c.set = set

	c.Unlock()

	// update watchers
	c.update()

	if len(gerr) > 0 {
		return fmt.Errorf("source loading errors: %s", strings.Join(gerr, "\n"))
	}

	return nil
}

// reload reads the sets and creates new values
func (c *config) reload() {
	c.Lock()

	// merge sets
	set, err := c.opts.Reader.Merge(c.sets...)
	if err != nil {
		c.Unlock()
		return
	}

	// set values
	c.vals, _ = c.opts.Reader.Values(set)
	c.set = set

	c.Unlock()

	// update watchers
	c.update()
}

func (c *config) Close() error {
	select {
	case <-c.exit:
		return nil
	default:
		close(c.exit)
	}
	return nil
}

func (c *config) Get(path ...string) reader.Value {
	if !c.loaded() {
		c.Sync()
	}

	c.Lock()
	defer c.Unlock()

	// did sync actually work?
	if c.vals != nil {
		return c.vals.Get(path...)
	}

	ch := c.set

	// we are truly screwed, trying to load in a hacked way
	v, err := c.opts.Reader.Values(ch)
	if err != nil {
		log.Printf("Failed to read values %v trying again", err)
		// man we're so screwed
		// Let's try hack this
		// We should really be better
		if ch == nil || ch.Data == nil {
			ch = &source.ChangeSet{
				Timestamp: time.Now(),
				Source:    "config",
				Data:      []byte(`{}`),
			}
		}
		v, _ = c.opts.Reader.Values(ch)
	}

	// lets set it just because
	c.vals = v

	if c.vals != nil {
		return c.vals.Get(path...)
	}

	// ok we're going hardcore now
	return newValue()
}

func (c *config) Bytes() []byte {
	if !c.loaded() {
		c.Sync()
	}

	c.Lock()
	defer c.Unlock()

	if c.vals == nil {
		return []byte{}
	}

	return c.vals.Bytes()
}

func (c *config) Load(sources ...source.Source) error {
	var gerrors []string

	for _, source := range sources {
		set, err := source.Read()
		if err != nil {
			gerrors = append(gerrors,
				fmt.Sprintf("error loading source %s: %v",
					source,
					err))
			// continue processing
			continue
		}
		c.Lock()
		c.sources = append(c.sources, source)
		c.sets = append(c.sets, set)
		idx := len(c.sets) - 1
		c.Unlock()
		go c.watch(idx, source)
	}

	c.reload()

	// Return errors
	if len(gerrors) != 0 {
		return errors.New(strings.Join(gerrors, "\n"))
	}
	return nil
}

func (c *config) Watch(path ...string) (Watcher, error) {
	value := c.Get(path...)

	c.Lock()

	w := &watcher{
		exit:    make(chan bool),
		path:    path,
		value:   value,
		updates: make(chan reader.Value, 1),
	}

	id := c.idx
	c.watchers[id] = w
	c.idx++

	c.Unlock()

	go func() {
		<-w.exit
		c.Lock()
		delete(c.watchers, id)
		c.Unlock()
	}()

	return w, nil
}

func (c *config) String() string {
	return "config"
}

func (w *watcher) Next() (reader.Value, error) {
	for {
		select {
		case <-w.exit:
			return nil, errors.New("watcher stopped")
		case v := <-w.updates:
			if bytes.Equal(w.value.Bytes(), v.Bytes()) {
				continue
			}
			w.value = v
			return v, nil
		}
	}
}

func (w *watcher) Stop() error {
	select {
	case <-w.exit:
	default:
		close(w.exit)
	}
	return nil
}
