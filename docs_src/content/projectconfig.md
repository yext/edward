---
date: 2017-02-01T17:24:55-05:00
title: Project Configuration
---

## edward.json

The *edward.json* file defines the services and groups that can be managed by Edward.

Edward will look for an *edward.json* file in the current working directory, and if not found there,
it will look for *edward.json* in every parent directory to the root.

If no config file can be found, Edward will exit with an error and print usage information.

You can override this behavior with the `--config` or `-c` flag:

    $ edward -c path/to/alternative.json list

## Structure

An *edward.json* file at its core is a JSON document, containing an object with three array attributes:

```json
{
  "imports": [],
  "groups": [],
  "services": []
}
```

These attributes are all optional.

## Services

A basic service consists of a *name*, a *path* in which to run the build and launch commands, and *commands*
to perform the build and launch steps:

```json
{
    "name": "myservice",
    "path": "path/to/myservice",
    "commands": {
        "build": "make install",
        "launch": "myservice"
    }
}
```

The service *name* is the name you will use to identify this service when calling Edward commands.

{{< warning title="Names must be unique" >}}
Service and group names must be unique within your Edward config. If duplicates are found, Edward will
exit with an error.
{{< /warning >}}

The *path* and each of the *commands* are optional. If no path is specified, Edward will run the commands in the current working directory. If a build command is omitted, only the launch script will be executed, and vice versa. This allows you to run pre-flight build steps common to many services, or start third-party applications you don't need to build.

You can use environment variables in the path and commands:

```json
{
    "name": "myservice",
    "path": "$HOME/src/myservice",
    "commands": {
        "launch": "myservice -key $MY_KEY"
    }
}
```

However, you cannot use shell-specific constructs like piping (`|`) or redirection (`>`). For complex
build and launch operations, we recommend consolidating multiple commands into a separate script file.

### Custom Stop Scripts

If you need to run a specific command to stop a service, you can add a *stop* command to the service definition:

```json
{
    "name": "myservice",
    "commands": {
        "launch": "myservice up",
        "stop": "myservice down"
    }
}
```

If this command fails, Edward will attempt to send a kill signal to the running service.

### Detecting Successful Launch

When Edward starts a service, it will confirm that the service has started successfully before proceeding. By default, Edward will consider a service to have started if it is listening on one or more ports.

You can override this behavior by setting the *launch_checks* attribute for the service.

If your service will output a known piece of text to the console when it has finished startup, you can specify a *log_text*:

```json
{
    "name": "myservice",
    ...
    "launch_checks": {
      "log_text": "Finished startup"
    }
}
```

In this case, once the text "Finished startup" appears in the console lot, Edward will deem the service to have started successfully.

Alternatively, you can specify a set of ports:

```json
{
    "name": "myservice",
    ...
    "launch_checks": {
      "ports": [8080, 8081]
    }
}
```

And Edward will wait for all the listed ports to be open before considering the service started. When ports are specified, the process that opens them will not be taken into account.

### Environment Variables

To specify environment variables to be passed to a service, you can add the *env* attribute, which is an array of environment variables in the form `KEY=VALUE`:

```json
{
    "name": "myservice",
    ...
    "env": [
      "ENV_VAR=value"
    ]
}
```

### Platform-Specific Services

Some services need different configuration for different platforms. To make a service platform-specific, set the *platform* attribute.

It is permitted to have multiple services with the same name, provided they have different platforms.

The below example will create two instances of *myservice*: one for Mac OS (darwin) and one for Linux:

```json
{
    "name": "myservice",
    ...
    "platform": "darwin"
},
{
    "name": "myservice",
    ...
    "platform": "linux"
}
```

### "Warming Up" Services

Some services may do a portion of their setup on the first request they receive. To cut down on waiting
time when working with such services, you can configure Edward to make a request to a URL after starting a service.

```json
{
    "name": "myservice",
    ...
    "warmup": {
      "url": "http://localhost:8080"
    }
}
```

Requests to these URLs will happen in the background, and will not delay the starting of other services.

### Requiring Sudo

If a service needs sudo to run, it will need to be marked appropriately:

```json
{
  "name": "myservice",
  "requiresSudo": true
}
```

If any service to be started/stopped/restarted requires sudo, Edward will trigger a prompt for the user's password.
This prompt is triggered through scripting, Edward itself will not have access to your password.

## Groups

The *groups* array contains a list of group objects, each of which have a name and a list of children.
A group's children can be either services, or other groups.

```json
{
  "name": "mygroup",
  "children": ["childgroup", "childservice"]
}
```

The above example specifies a group called *mygroup* with two children, *childgroup* and *childservice*.

## Imports

The *imports* array is a list of other config files to be imported into this one:

```json
"imports": ["import1.json", "path/to/import2.json"]
```

The paths to imports are relative to the parent `edward.json` file. Imported config files may also import other
config files.

The combined configuration is validated after all imports have been loaded, so a group in one file may have as a child a service from another file, provided they are connected by an import in some way.

## Versioning

If you are using features from a new version of Edward, and want to make sure that your config file can only be used by that version or higher, you can specify the *edwardVersion* setting in your config file:

```json
{
  "edwardVersion": "1.6.0",
  "imports": [],
  "groups": [],
  "services": []
}
```

This will require that Edward version 1.6.0 or higher is installed in order to use your config file.
