package main

import (
	"context"
	"log"
	"time"

	"github.com/flowerinthenight/kettle/v2"
)

type app struct {
	K    *kettle.Kettle
	Name string
}

func (a *app) DoMaster(v interface{}) error {
	k := v.(*kettle.Kettle)
	log.Printf("[%v] HELLO FROM MASTER", k.Name())
	return nil
}

func (a app) DoWork() error {
	log.Printf("[%v] hello from worker, master=%v", a.Name, a.K.IsMaster())
	return nil
}

func main() {
	// Our app object abstraction.
	name := "kettle-simple-example"
	a := &app{Name: name}
	k, err := kettle.New(kettle.WithName(name), kettle.WithVerbose(true))
	if err != nil {
		log.Fatal(err)
	}

	a.K = k // store reference to kettle
	quit, cancel := context.WithCancel(context.TODO())
	done := make(chan error, 1)
	in := kettle.StartInput{
		Master:    a.DoMaster, // called when we are master
		MasterCtx: k,          // context value that is passed to `Master` as parameter
	}

	err = k.Start(quit, &in, done) // start kettle
	if err != nil {
		log.Fatal(err)
	}

	// Proceed with our usual (simulated) worker job.
	for i := 0; i < 20; i++ {
		a.DoWork()
		time.Sleep(time.Second * 2)
	}

	cancel() // terminate
	<-done   // wait
}
