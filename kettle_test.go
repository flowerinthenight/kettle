package kettle

import (
	"os"
	"testing"
	"time"
)

func TestGen(t *testing.T) {
	if host := os.Getenv("REDIS_HOST"); host == "" {
		t.Log("no redis host:port")
		return
	}

	k, err := New(WithName("kettle_v0"), WithVerbose(true), WithTickTime(5))
	if err != nil {
		t.Fatal(err)
	}

	in := StartInput{
		Master: func(v interface{}) error {
			kt := v.(*Kettle)
			t.Log("from master, name:", kt.Name())
			return nil
		},
		MasterCtx: k,
		Quit:      make(chan error, 1),
		Done:      make(chan error, 1),
	}

	err = k.Start(&in)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 3)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
