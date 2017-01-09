---
date: 2016-03-09T00:11:02+01:00
title: Getting started
weight: 10
---

## Setup

Before starting with these instructions, make sure you've [installed Edward](/installation/).

To get you started with Edward, we're going to create a few simple services
and groups. Our example services will all be written in Go.

These instructions assume a bash terminal.

First off, let's create a parent folder for our services. This is where the Edward
config file will live. In your *GOPATH*, create a folder called *edward_quickstart*:

```sh
mkdir $GOPATH/edward_quickstart
```

If you prefer, you can also create this folder under your GitHub username, to fit
in with your other projects: *$GOPATH/github.com/user/edward_quickstart*. The
following instructions will assume *$GOPATH/edward_quickstart* for brevity.

## Creating Your First Service

Your first service will be a classic "Hello, world!" HTTP server.
This service will sit under *edward_quickstart* and will consist of a single file,
*main.go*. Let's create that directory and file now:

```sh
mkdir $GOPATH/edward_quickstart/hello
touch $GOPATH/edward_quickstart/hello/main.go
```

Open the main.go file you just created and add this code:

```go
package main

import (
	"flag"
	"fmt"
	"net/http"
)

var port = flag.Int("port", 8080, "Port number for service")

func main() {
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world!")
	})
	http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)
}
```

This service will listen on port 8080 by default, and return "Hello, world!" in
the body of every response. But we're not going to run it quite yet, first we're
going to generate a config so we can launch this service with Edward.

## Generating an Edward Config

Edward includes a generator for go services which will automatically create config
for any go services found under the current working directory. If we run `edward generate`
it will create a config for our new service:

```sh
cd $GOPATH/edward_quickstart/hello
edward generate
```

You will see a message letting you know that config for the service *hello* will
be generated and you will be asked to confirm. Type `y` to confirm.

A file called *edward.json* will be created in the *hello* directory. You can check
that *hello* is available as a service by listing all available services.

```sh
edward list
```

You should see a list that only includes the *hello* service.

## Starting your Service

Now we have our configuration set up, we can run the hello service with this command:

```sh
edward start hello
```

This will build and launch the hello service. Once the command completes, open
a browser and go to *http://localhost:8080/*, you should see the message
**Hello, world!**.

## Stopping your Service

Your service is now running in the background, you can stop it with the `stop` command:

```sh
edward stop hello
```

If you now try to browse to *http://localhost:8080/*, you will see an error message,
as the server is no longer running.

When running more than one service, you can stop all services at once with `edward stop`.

## Restarting your Service

If you've made changes to a service, you can restart it to trigger a rebuild.
To try this out, first start the service again.

```sh
edward start hello
```

Now, while the service is running, open the *main.go* file you created earlier, and change:

```go
fmt.Fprintf(w, "Hello, world!")
```

to:

```go
fmt.Fprintf(w, "Greetings, world!")
```

To apply this change to your service, run the command:

```sh
edward restart hello
```

This will stop the service, then build and launch it again. If you browse to
*http://localhost:8080/* again, you will see that the message has changed to *Greetings, world!*.

## Modifying the Edward Config

The Edward config you generated can be found at *$GOPATH/edward_quickstart/edward.json*.
We're going to edit this file to have the hello service listen on a different port.

Open *edward.json*, you will see the hello service defined as the only entry in the *services* array.
Under *commands*, you will see two string values, *build* and *launch*.
These define the commands used to build and launch your service, respectively.

We're going to change the launch command to use a different port. Change:

```json
"launch": "hello"
```

To:

```json
"launch": "hello -port=8081"
```

Save the config file, and restart the hello service as you did earlier.

The URL *http://localhost:8080/* will now no longer work, but *http://localhost:8081/*
will show the message as expected.

## Adding a Second Service

To complement the hello service, let's add a goodbye service. First, create a directory
and source file:

```sh
mkdir $GOPATH/edward_quickstart/goodbye
touch $GOPATH/edward_quickstart/goodbye/main.go
```

Edit *$GOPATH/edward_quickstart/goodbye/main.go* and add the following code:

```go
package main

import (
	"flag"
	"fmt"
	"net/http"
)

var port = flag.Int("port", 8082, "Port number for service")

func main() {
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Goodbye, world!")
    fmt.Println("Received request")
	})
	http.ListenAndServe(fmt.Sprintf(":%v", *port), nil)
}
```

If you run `edward generate` now, it will create the goodbye service (you may be
prompted for both services, don't worry, just confirm), but the hello service's
config will not be changed.

Run `edward list` to confirm that both services are available.

We can start our new service in the same way we started hello:

```sh
edward start goodbye
```

And can view the goodbye message at *http://localhost:8082/*.

We can also start both services at once. First, stop everything by running `edward stop`.
Then run a start command specifying both services.

```sh
edward start hello goodbye
```

Both services will now be running. You can test at their URLs, but you can also
see a summary of all your running services with:

```sh
edward status
```

This will show that both services are running, along with the port on which they are running, which
should be 8081 for hello and 8082 for goodbye.

## Creating a Group

Specifying every service you want to start can be useful for one or two services,
but can get tedious for more complicated setups.

To simplify the workflow, you can create groups of services which can be started
all at once from a single name.

Open your *edward.json* file and add a *groups* array as follows:

```json
{
    "groups": [
      {
        "name": "salutations",
        "children": ["hello", "goodbye"]
      }
    ],
    "services": [
      ...
    ]
}
```

Save the file and you will now see *salutations* under the groups list when
you run `edward list`.

Stop any running services, then start the group:

```sh
edward start salutations
```

This will build and launch both services, which you can confirm with `edward status`.

## Viewing Logs

For debugging, you'll need to be able to view the output from your services. You can
follow the output from one service using the `log` command.

Make sure your services are still running, and run the log command to see the output for
the goodbye service:

```sh
edward log goodbye
```

This will begin following the output from goodbye. If you visit *http://localhost:8082/*,
you will see the message "Received request" in the output. You can stop following
the log by pressing `Ctrl+c` to interrupt the command.

Running `edward log` will show both standard and error output. You can also use
the alias `edward tail`.

## Summary

After this guide, you should now be able to use Edward to:

* Start, stop and restart services and groups
* Generate config for go services
* Set up groups
* View service status
* Follow service output
