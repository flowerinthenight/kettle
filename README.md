# Overview
`kettle` is a simple library that abstracts the use of distributed locking to elect a master among group of workers at a specified time interval. The elected master will then call the "master" function. This library is developed with containers in mind and uses [Redis](https://redis.io/) as the default distributed locker.

# How it works
