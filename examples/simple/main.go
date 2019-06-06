package main

import (
	"log"
	"time"

	"github.com/flowerinthenight/kettle"
)

type app struct {
	Value int32
}

func (w *app) DoMaster(v interface{}) error {
	k := v.(*kettle.Kettle)
	log.Printf("[%v] HELLO FROM MASTER", k.Name())
	return nil
}

func (w *app) DoWork() error {
	return nil
}

func main() {
	// Our app object abstraction.
	w := &app{}

	name := "kettle-example"
	k, err := kettle.New(
		kettle.WithName(name),
		kettle.WithVerbose(true),
	)

	if err != nil {
		log.Fatal(err)
	}

	in := kettle.StartInput{
		Master:    w.DoMaster,       // called when we are master
		MasterCtx: k,                // context value that is passed to `Master` as parameter
		Quit:      make(chan error), // tell kettle to exit
		Done:      make(chan error), // kettle is done
	}

	err = k.Start(&in) // start kettle
	if err != nil {
		log.Fatal(err)
	}

	// Proceed with normal worker job.
	go func() {
		for {
			log.Printf("[%v] hello from worker", name)
			time.Sleep(time.Second * 2)
		}
	}()

	time.Sleep(time.Second * 40)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
