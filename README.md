# Edward

[![Build Status](https://travis-ci.org/yext/edward.svg?branch=master)](https://travis-ci.org/yext/edward)
[![Go Report Card](https://goreportcard.com/badge/github.com/yext/edward)](https://goreportcard.com/report/github.com/yext/edward)

A command line tool for managing local instances of microservices.

## Installation

Edward uses the vendor folder, and as such, Go 1.6 is required.

    go get github.com/yext/edward

## Updating

To update an existing install to the latest version of Edward, run:

    go get -u github.com/yext/edward

## Usage

    NAME:
       Edward - Manage local microservices

    USAGE:
       edward [global options] command [command options] [arguments...]

    VERSION:
       1.6.4

    COMMANDS:
         list	List available services
         generate	Generate Edward config for a source tree
         status	Display service status
         messages	Show messages from services
         start	Build and launch a service
         stop	Stop a service
         restart	Rebuild and relaunch a service
         log, tail	Tail the log for a service

    GLOBAL OPTIONS:
       --config PATH, -c PATH	Use service configuration file at PATH
       --help, -h			show help
       --version, -v		print the version

## Documentation

Full documentation is available at [http://engblog.yext.com/edward/](http://engblog.yext.com/edward/).
