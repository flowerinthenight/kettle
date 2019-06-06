# Overview
`kettle` is a simple library that abstracts the use of distributed locking to elect a master among group of workers at a specified time interval. The elected master will then call the "master" function. This library uses [Redis](https://redis.io/) as the default [distributed locker](https://redis.io/topics/distlock).

# How it works
All workers that share the same name will attempt to grab a Redis lock to become the master. A provided master function will be executed by the node that successfully grabbed the lock. A single node works as well, in which case, that node will run both as master and a worker.

```go
  name := "kettle-example"
  k, _ := kettle.New(
    kettle.WithName(name),
    kettle.WithVerbose(true),
  )
  
  in := kettle.StartInput{
    // Our master callback function.
    Master: func(v interface{}) error {
      kt := v.(*kettle.Kettle)
      log.Println("from master, name:", kt.Name())
      return nil
    },
    MasterCtx: k, // arbitrary data that is passed to master function
    Quit:      make(chan error),
    Done:      make(chan error),
  }

  err = k.Start(&in)
  _ = err

  // Simulate work
  time.Sleep(time.Second * 5)

  in.Quit <- nil // terminate
  <-in.Done      // wait
```
