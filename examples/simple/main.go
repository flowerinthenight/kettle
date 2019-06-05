package main

import (
	"log"
	"time"

	"github.com/flowerinthenight/kettle"
)

func doMaster(v interface{}) error {
	c := v.(string)
	log.Println("doing master work with context:", c)
	return nil
}

func main() {
	s, err := kettle.New(kettle.WithVerbose(true))
	if err != nil {
		log.Fatal(err)
	}

	in := kettle.StartInput{
		Master:    doMaster,
		MasterCtx: "masterctx",
		Quit:      make(chan error),
		Done:      make(chan error),
	}

	err = s.Start(&in)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 40)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
