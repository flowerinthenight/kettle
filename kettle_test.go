package kettle

import (
	"log"
	"testing"
	"time"
)

func TestGen(t *testing.T) {
	s, err := New(withVerbose(true))
	if err != nil {
		t.Fatal(err)
	}

	in := StartInput{
		Master: func(v interface{}) error {
			c := v.(string)
			log.Println(c)
			return nil
		},
		MasterCtx: "masterctx",
		Quit:      make(chan error),
		Done:      make(chan error),
	}

	err = s.Start(&in)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 40)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
