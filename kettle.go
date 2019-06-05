package kettle

import (
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

type DistLocker interface {
	Lock() error
	Unlock() bool
}

type KettleOption interface {
	Apply(*kettle)
}

type withName string

func (w withName) Apply(o *kettle)   { o.name = string(w) }
func WithName(v string) KettleOption { return withName(v) }

type withVerbose bool

func (w withVerbose) Apply(o *kettle) { o.verbose = bool(w) }
func WithVerbose(v bool) KettleOption { return withVerbose(v) }

type withDistLocker struct{ dl DistLocker }

func (w withDistLocker) Apply(o *kettle)       { o.lock = w.dl }
func WithDistLocker(v DistLocker) KettleOption { return withDistLocker{v} }

type withTickTime int64

func (w withTickTime) Apply(o *kettle)  { o.tickTime = int64(w) }
func WithTickTime(v int64) KettleOption { return withTickTime(v) }

type kettle struct {
	name       string
	verbose    bool
	pool       *redis.Pool
	lock       DistLocker
	master     int32 // 1 if we are master, otherwise, 0
	hostname   string
	startInput *StartInput // copy of StartInput
	masterQuit chan error  // signal master set to quit
	masterDone chan error  // master termination done
	tickTime   int64
}

func (s kettle) Name() string      { return s.name }
func (s kettle) IsVerbose() bool   { return s.verbose }
func (s kettle) Pool() *redis.Pool { return s.pool }

func (s kettle) info(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s kettle) infof(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s kettle) error(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s kettle) errorf(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s kettle) fatal(v ...interface{}) {
	s.error(v...)
	os.Exit(1)
}

func (s kettle) fatalf(format string, v ...interface{}) {
	s.errorf(format, v...)
	os.Exit(1)
}

func (s kettle) isMaster() bool {
	if atomic.LoadInt32(&s.master) == 1 {
		return true
	} else {
		return false
	}
}

func (s *kettle) setMaster() {
	if err := s.lock.Lock(); err != nil {
		s.infof("[%v] %v set to worker", s.name, s.hostname)
		atomic.StoreInt32(&s.master, 0)
		return
	}

	s.infof("[%v] %v set to master", s.name, s.hostname)
	atomic.StoreInt32(&s.master, 1)
}

func (s *kettle) doMaster() {
	masterTicker := time.NewTicker(time.Second * time.Duration(s.tickTime))

	work := func() {
		// Attempt to be master here.
		s.setMaster()

		// Only if we are master.
		if s.isMaster() {
			if s.startInput.Master != nil {
				s.startInput.Master(s.startInput.MasterCtx)
			}
		}
	}

	work() // first invoke before tick

	go func() {
		for {
			select {
			case <-masterTicker.C:
				work() // succeeding ticks
			case <-s.masterQuit:
				s.masterDone <- nil
				return
			}
		}
	}()
}

type StartInput struct {
	Master    func(ctx interface{}) error // function to call every time we are master
	MasterCtx interface{}                 // callback function parameter
	Quit      chan error                  // signal for us to terminate
	Done      chan error                  // report that we are done
}

func (s *kettle) Start(in *StartInput) error {
	if in == nil {
		return errors.Errorf("input cannot be nil")
	}

	s.startInput = in
	hostname, _ := os.Hostname()
	hostname = hostname + fmt.Sprintf("__%s", uuid.NewV4())
	s.hostname = hostname

	s.masterQuit = make(chan error, 1)
	s.masterDone = make(chan error, 1)

	go func() {
		<-in.Quit
		s.infof("[%v] requested to terminate", s.name)

		// Attempt to gracefully terminate master.
		s.masterQuit <- nil
		<-s.masterDone

		s.infof("[%v] terminate complete", s.name)
		in.Done <- nil
	}()

	go s.doMaster()

	return nil
}

func New(opts ...KettleOption) (*kettle, error) {
	s := &kettle{
		name:     "kettle",
		tickTime: 30,
	}

	for _, opt := range opts {
		opt.Apply(s)
	}

	if s.lock == nil {
		pool, err := NewRedisPool()
		if err != nil {
			return nil, err
		}

		s.pool = pool
		pools := []redsync.Pool{pool}
		rs := redsync.New(pools)
		s.lock = rs.NewMutex(fmt.Sprintf("%v-distlocker", s.name), redsync.SetExpiry(time.Second*time.Duration(s.tickTime)))
	}

	return s, nil
}
