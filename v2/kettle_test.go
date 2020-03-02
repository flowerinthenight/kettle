package kettle

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestGen(t *testing.T) {
	if host := os.Getenv("REDIS_HOST"); host == "" {
		log.Println("no redis host:port")
		return
	}

	k, err := New(withVerbose(true))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.TODO())
	done := make(chan error, 1)
	in := StartInput{
		Master: func(v interface{}) error {
			kt := v.(*Kettle)
			log.Println("from master, name:", kt.Name())
			return nil
		},
		MasterCtx: k,
	}

	err = k.Start(ctx, &in, done)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	cancel() // terminate
	<-done   // wait
}
