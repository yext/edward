# Edward

*Edward is no longer being actively maintained*. We recommend using [Tilt](https://tilt.dev/) for this and similar use cases.

![Tests](https://github.com/yext/edward/workflows/Test/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/yext/edward)](https://goreportcard.com/report/github.com/yext/edward)
[![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/yext-edward/Lobby)

A command line tool for managing local instances of microservices.

[![asciicast](https://asciinema.org/a/c49y8xmkvv140rgnlvl0an908.png)](https://asciinema.org/a/c49y8xmkvv140rgnlvl0an908?autoplay=1)

Full documentation available at [http://engblog.yext.com/edward/](http://engblog.yext.com/edward/).

## Table of Contents  

* [Features](#features)
* [Installation](#installation)  
* [Updating](#updating)

## Features

### Start multiple services with one command

No need to start each service in its own terminal tab, just run `edward start` to build and launch multiple
services in the background!

[See it in action](https://asciinema.org/a/c49y8xmkvv140rgnlvl0an908?autoplay=1)

### See status for running services

Run `edward status` to see which of your services are up and running, how long for, and on which ports
they are listening.

[See it in action](https://asciinema.org/a/c49y8xmkvv140rgnlvl0an908?t=10&autoplay=1)

### Follow service logs

Follow stdout and stderr for one or more services with `edward tail`.

[See it in action](https://asciinema.org/a/5yt0iobii6f62swt4l67sm513?autoplay=1)

### Restart as needed

Made some changes? Run `edward restart` to re-build and re-launch a service.

[See it in action](https://asciinema.org/a/0epxufbswt2c8vf8lw10g92qo?autoplay=1)

### Auto-restart on edits

Edward will even automatically restart services when source files are changed.

[See it in action](https://asciinema.org/a/7shqwxugaxxstccyry6c8ox2r?autoplay=1)

### Generate configuration automatically

New services? Run `edward generate` to create a config file automatically.

[See it in action](https://asciinema.org/a/dhbv5v0dpd65sgfjud92auyt0?autoplay=1)

Edward can generate configuration for projects using:

* Go
* Docker
* ICBM
* Procfiles
* Other Edward config files

Don't see your project described above? No problem! Edward can be manually configured for any
service that can be built and started from the command line.

## Installation

Edward requires Mac OS or Linux and Go 1.7 or higher.

    go get github.com/yext/edward

## Updating

To update an existing install to the latest version of Edward, run:

    go get -u github.com/yext/edward
