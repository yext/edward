---
date: 2017-02-01T17:24:55-05:00
title: Commands
---

Edward provides a series of commands to manage your local services, which are called
using the form:

    $ edward [command]

You may have discovered some of these already in the [Quickstart](../quickstart/),
but they are outlined in more detail here.

## List

The `list` command outputs a list of all the services and groups that are defined
in the current [`edward.json`](../projectconfig). It takes no arguments.

Groups are listed first, followed by services. The list will include all services
that are defined by your `edward.json` or any of the files it imports.

## Start

The `start` command will build and launch a service or group locally. It takes
the name of one or more services as arguments.

For example, to start the service named *myservice*, you would run:

    $ edward start myservice

To start the group *mygroup* along with *myservice*, you would run:

    $ edward start mygroup myservice

When starting multiple services, it will build and launch them sequentially.

If any of the specified services are already running, Edward will ignore them and move on to the next
service.

If Edward fails to build or launch any services, it will stop the operation and any subsequent specified
services will not start.

Once all services have been started, the Edward command will exit, and these services will continue to run in the background.

## Stop

The `stop` command will stop one or more groups and/or services. It takes service
and group names are arguments in the same manner as `start`.

For example, to stop *mygroup* and *myservice*:

    $ edward stop mygroup myservice

If any of the specified services are not running, Edward will ignore them and move on to the next
service. If Edward fails to stop a service, it will continue to stop the rest of the services specified.

## Restart

The `restart` command will rebuild and relaunch the specified groups/services.  It takes service
and group names are arguments in the same manner as `start`.

To restart *mygroup*:

    $ edward restart mygroup

Each service will be stopped, rebuilt and relaunched sequentially. If Edward fails to start, build or launch any
service, the operation will end, as with `start`.

## Log/Tail

The `log` or `tail` command will output and then follow the console logs for the specified groups/services.

For example, to output and follow the logs for *myservice*:

    $ edward log myservice

Or:

    $ edward tail myservice

If more than one service is being output, the name of the service will be added to the start
of each line in the log to distinguish them.

## Generate

The `generate` command will search in the current working directory for projects for which Edward
can automatically generate a config file. It will then either create a new `edward.json` file in the
working directory, or add to the existing file.

For example:

    $ edward generate

Will search in all directories below the working directory and automatically generate config for any
supported projects that are found.

If you want to only generate config for a specific project, you can specify it as an argument:

   $ edward generate myservice

Edward supports autogeneration for three types of project:

* go
* Docker
* icbm

### Go

The *Go* generator will create service configuration for services written in the [Go programming language](https://golang.org/).

This generator will match any folder containing a `main` package. The name of this
folder will be used as the name of the service.

The generated config will build the project by changing into the package directory and running `go install`
with no additional arguments. The name of the project will be used to launch the resulting binary.

The generated config will assume that a service has started successfully by detecting that it is listening on
at least one port. If a service does not listen on any ports, it will time out when starting.

The *Go* generator will attempt to configure the watch paths for Go services, based on the imports of the project's source.

Note that this generator assumes that your *GOPATH* is configured correctly to build discovered projects.

Once a Go service has been found, any folders inside the package directory will not be searched.

### Docker

The *Docker* generator will create service configuration for [Docker containers](https://www.docker.com/).

This generator will match any folder containing a *Dockerfile*. The name of this folder
will be used as the name of the service.

The generated config will build the container using `docker build` and start it using `docker run`. An
Edward specific tag will be used to identify these container instances.

The port to be used for this service will be identified by the `EXPOSE` command in the Dockerfile. This same port will be opened locally for this service. Edward will identify that this container has started successfully when the
exposed port is open. If `EXPOSE` is not used, starting the container will time out.

Note that this generator will assume that you can execute the `docker` command without additional configuration, so older Docker Toolkit distributions may not work.

### icbm

The *icbm* generator will generate service configuration for services that use the [icbm](https://github.com/yext/icbm) build tool.

This generator will look for a *build.spec* file and generate a service for each of the named aliases.

The generated config will assume that a service has started successfully by detecting that it is listening on
at least one port. If a service does not listen on any ports, it will time out when starting.

### Ignoring directories

To protect against false positives, you can instruct Edward to ignore specific patterns when running `generate` by creating an *.edwardignore* file.

This file uses the same format as [gitignore](https://git-scm.com/docs/gitignore). You can place an *.edwardignore* file in any directory and it will take effect for paths below that directory, replacing ignores specified by ignore files higher up.
