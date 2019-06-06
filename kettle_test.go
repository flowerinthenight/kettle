package kettle

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestGen(t *testing.T) {
	if host := os.Getenv("REDIS_HOST"); host == "" {
		log.Println("ho redis host:port")
		return
	}

	k, err := New(withVerbose(true))
	if err != nil {
		t.Fatal(err)
	}

	in := StartInput{
		Master: func(v interface{}) error {
			kt := v.(*Kettle)
			log.Println("from master, name:", kt.Name())
			return nil
		},
		MasterCtx: k,
		Quit:      make(chan error),
		Done:      make(chan error),
	}

	err = k.Start(&in)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
