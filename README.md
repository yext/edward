# Edward

[![Build Status](https://travis-ci.org/yext/edward.svg?branch=master)](https://travis-ci.org/yext/edward)
[![Go Report Card](https://goreportcard.com/badge/github.com/yext/edward)](https://goreportcard.com/report/github.com/yext/edward)

A command line tool for managing local instances of microservices.

## Start multiple services with one command

No need to start each service in its own terminal tab, just run `edward start` to build and launch multiple
services in the background!

![Starting services](images/start.gif)

## See status for running services

Run `edward status` to see which of your services are up and running, how long for, and on which ports
they are listening.

![View Status](images/status.gif)

## Follow service logs

Follow stdout and stderr for one or more services with `edward tail`.

![Follow logs](images/tail.gif)

## Restart as needed

Made some changes? Run `edward restart` to re-build and re-launch a service.

![Restart services](images/restart.gif)

## Auto-restart on edits

Edward will even automatically restart services when source files are changed.

![Auto-restart when files are edited](images/autorestart.gif)

## Generate configuration automatically

New services? Run `edward generate` to create a config file automatically.

![Generate configuration](images/generate.gif)

Edward can generate configuration for:

* Go
* Docker
* ICBM

Using a different language? No problem! Edward can be manually configured for any
service that can be built and started from the command line.

## Installation

Edward uses the vendor folder, and as such, Go 1.6 is required.

    go get github.com/yext/edward

## Updating

To update an existing install to the latest version of Edward, run:

    go get -u github.com/yext/edward

## Documentation

Full documentation is available at [http://engblog.yext.com/edward/](http://engblog.yext.com/edward/).
