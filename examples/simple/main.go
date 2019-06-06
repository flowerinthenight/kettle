package main

import (
	"log"
	"time"

	"github.com/dchest/uniuri"
	"github.com/flowerinthenight/kettle"
)

type work struct {
	Value int32
}

func (w *work) DoMaster(v interface{}) error {
	k := v.(*kettle.Kettle)
	log.Printf("[%v] hello from master", k.Name())
	return nil
}

func (w *work) DoWork() error {
	return nil
}

func main() {
	// Our worker object abstraction.
	w := &work{
		Value: 1, // this will be incremented every time master runs
	}

	name := uniuri.NewLen(10)
	// Setup kettle for master work.
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
			log.Printf("[%v] i am doing worker job here", name)
			time.Sleep(time.Second * 2)
		}
	}()

	time.Sleep(time.Second * 40)
	in.Quit <- nil // terminate
	<-in.Done      // wait
}
