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
	// Setup what to do when we are master.
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

	// Proceed with normal worker job.
	go func() {
		for {
			log.Println("i am doing worker job here")
			time.Sleep(time.Second * 2)
		}
	}()

	time.Sleep(time.Second * 40)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
