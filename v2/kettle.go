package kettle

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

// DistLocker abstracts a distributed locker.
type DistLocker interface {
	Lock() error
	Unlock() (bool, error)
}

// KettleOption configures Kettle.
type KettleOption interface {
	Apply(*Kettle)
}

type withName string

// Apply applies a name to a Kettle instance.
func (w withName) Apply(o *Kettle) { o.name = string(w) }

// WithName configures Kettle instance's name.
func WithName(v string) KettleOption { return withName(v) }

type withVerbose bool

// Apply applies a verbosity value to a Kettle instance.
func (w withVerbose) Apply(o *Kettle) { o.verbose = bool(w) }

// WithVerbose configures a Kettle instance's log verbosity.
func WithVerbose(v bool) KettleOption { return withVerbose(v) }

type withDistLocker struct{ dl DistLocker }

// Apply applies a distributed locker to a Kettle instance.
func (w withDistLocker) Apply(o *Kettle) { o.lock = w.dl }

// WithDistLocker configures a Kettle instance's DistLocker.
func WithDistLocker(v DistLocker) KettleOption { return withDistLocker{v} }

type withTickTime int64

// Apply applies a tick time interval value to a Kettle instance.
func (w withTickTime) Apply(o *Kettle) { o.tickTime = int64(w) }

// WithTickTime configures a Kettle instance's tick timer in seconds.
func WithTickTime(v int64) KettleOption { return withTickTime(v) }

// Kettle provides functions that abstract the master election of a group of workers
// at a given interval time.
type Kettle struct {
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

// Name returns the instance's name.
func (k Kettle) Name() string { return k.name }

// IsVerbose returns the verbosity setting.
func (k Kettle) IsVerbose() bool { return k.verbose }

// IsMaster returns master status.
func (k Kettle) IsMaster() bool { return k.isMaster() }

// Pool returns the configured Redis connection pool.
func (k Kettle) Pool() *redis.Pool { return k.pool }

func (k Kettle) info(v ...interface{}) {
	if !k.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (k Kettle) infof(format string, v ...interface{}) {
	if !k.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (k Kettle) error(v ...interface{}) {
	if !k.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (k Kettle) errorf(format string, v ...interface{}) {
	if !k.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (k Kettle) fatal(v ...interface{}) {
	k.error(v...)
	os.Exit(1)
}

func (k Kettle) fatalf(format string, v ...interface{}) {
	k.errorf(format, v...)
	os.Exit(1)
}

func (k Kettle) isMaster() bool {
	if atomic.LoadInt32(&k.master) == 1 {
		return true
	} else {
		return false
	}
}

func (k *Kettle) setMaster() {
	if err := k.lock.Lock(); err != nil {
		atomic.StoreInt32(&k.master, 0)
		return
	}

	k.infof("[%v] %v set to master", k.name, k.hostname)
	atomic.StoreInt32(&k.master, 1)
}

func (k *Kettle) doMaster() {
	masterTicker := time.NewTicker(time.Second * time.Duration(k.tickTime))

	f := func() {
		// Attempt to be master here.
		k.setMaster()

		// Only if we are master.
		if k.isMaster() {
			if k.startInput.Master != nil {
				k.startInput.Master(k.startInput.MasterCtx)
			}
		}
	}

	f() // first invoke before tick

	go func() {
		for {
			select {
			case <-masterTicker.C:
				f() // succeeding ticks
			case <-k.masterQuit:
				k.masterDone <- nil
				return
			}
		}
	}()
}

// StartInput configures the Start function.
type StartInput struct {
	Master    func(ctx interface{}) error // function to call every time we are master
	MasterCtx interface{}                 // callback function parameter
}

// Start starts Kettle's main function. The ctx parameter is mainly used for termination
// with an optional done channel for us to notify when we are done, if any.
func (k *Kettle) Start(ctx context.Context, in *StartInput, done ...chan error) error {
	if in == nil {
		return fmt.Errorf("input cannot be nil")
	}

	k.startInput = in
	hostname, _ := os.Hostname()
	hostname = hostname + fmt.Sprintf("__%s", uuid.NewV4())
	k.hostname = hostname

	k.masterQuit = make(chan error, 1)
	k.masterDone = make(chan error, 1)

	go func() {
		<-ctx.Done()
		k.masterQuit <- nil
		<-k.masterDone
		if len(done) > 0 {
			done[0] <- nil
		}
	}()

	go k.doMaster()

	return nil
}

// New returns an instance of Kettle.
func New(opts ...KettleOption) (*Kettle, error) {
	k := &Kettle{name: "kettle", tickTime: 30}
	for _, opt := range opts {
		opt.Apply(k)
	}

	if k.lock == nil {
		pool, err := NewRedisPool()
		if err != nil {
			return nil, fmt.Errorf("NewRedisPool failed: %w", err)
		}

		k.pool = pool
		pools := []redsync.Pool{pool}
		rs := redsync.New(pools)
		k.lock = rs.NewMutex(
			fmt.Sprintf("%v-distlocker", k.name),
			redsync.SetExpiry(time.Second*time.Duration(k.tickTime)),
		)
	}

	return k, nil
}
