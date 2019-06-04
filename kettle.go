package share

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

const (
	masterTimeout = 30 // seconds
	keyTTL        = 29 // seconds

	cmdReportWorkerName = "CMD_REPORT_WORKER_NAME"
	cmdStartWork        = "CMD_START_WORK"
)

var (
	red   = color.New(color.FgRed).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
)

type DistLocker interface {
	Lock() error
	Unlock() error
}

type ShareOption interface {
	Apply(*share)
}

type withName string

func (w withName) Apply(o *share)   { o.name = string(w) }
func WithName(v string) ShareOption { return withName(v) }

type withVerbose bool

func (w withVerbose) Apply(o *share) { o.verbose = bool(w) }
func WithVerbose(v bool) ShareOption { return withVerbose(v) }

type withDistLocker struct{ dl DistLocker }

func (w withDistLocker) Apply(o *share)       { o.distlock = w.dl }
func WithDistLocker(v DistLocker) ShareOption { return withDistLocker{v} }

type share struct {
	name     string
	verbose  bool
	distlock DistLocker
}

func (s share) Name() string    { return s.name }
func (s share) IsVerbose() bool { return s.verbose }

func (s share) info(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s share) infof(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", green("[info]"), m)
}

func (s share) error(v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintln(v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s share) errorf(format string, v ...interface{}) {
	if !s.verbose {
		return
	}

	m := fmt.Sprintf(format, v...)
	log.Printf("%s %s", red("[error]"), m)
}

func (s share) fatal(v ...interface{}) {
	s.error(v...)
	os.Exit(1)
}

func (s share) fatalf(format string, v ...interface{}) {
	s.errorf(format, v...)
	os.Exit(1)
}

func (s share) InitRedisPool() *redis.Pool {
	addr := os.Getenv("REDIS_HOST")
	if addr == "" {
		s.fatal("REDIS_HOST env variable must be set (e.g host:port, redis://password@host:port)")
	}

	var dialOpts []redis.DialOption
	password := os.Getenv("REDIS_PASSWORD")
	if password != "" {
		dialOpts = append(dialOpts, redis.DialPassword(password))
	}

	timeoutEnv := os.Getenv("REDIS_TIMEOUT_SECONDS")
	if timeoutEnv != "" {
		timeoutSecs, timeoutParseErr := strconv.Atoi(timeoutEnv)
		if timeoutParseErr != nil {
			s.error("warning, REDIS_TIMEOUT_SECONDS must be a number")
		} else {
			dialOpts = append(dialOpts, redis.DialConnectTimeout(time.Duration(timeoutSecs)*time.Second))
		}
	}

	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   4,
		Wait:        true,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr, dialOpts...) },
	}
}

func New(dl DistLocker, opts ...ShareOption) (*share, error) {
	s := &share{
		name: fmt.Sprintf("%s", uuid.NewV4()),
	}

	for _, opt := range opts {
		opt.Apply(s)
	}

	pool := s.InitRedisPool()
	con := pool.Get()
	s.info(con)
	defer con.Close()

	return s, nil
}
