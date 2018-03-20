---
date: 2017-02-01T17:24:55-05:00
title: Backends (experimental)
---

Docker 1.8.9 introduces an experimental new feature: Backends.
A backend provides the functionality to build and run services on a particular platform.
A single service may specify multiple backends to be used in different situations.

Edward has built-in backends for the command line and Docker.

{{< warning title="Experimental feature" >}}
Backends are an experimental feature, and may be subject to breaking changes between releases of Edward.
{{< /warning >}}

## Configuration

Backends are specified in an array under a service definition.
Each backend is specified by a type, and a set of fields providing configuration for that backend.

```json
{
    "name": "myservice",
    "path": "path/to/myservice",
    "backends": [
        {
            "type": "<backend type>",
            "<field1>": "<value1>",
            "<field2>": "<value2>"
        }
    ]
}
```

## Command line

The command line backend provides the same functionality as detailed in the [Project Configuration](../projectconfig/#services) section.

Command line backends have the `type` "commandline", and accept the `commands` and `launch_checks` properties:

```json
{
    "name": "myservice",
    "path": "path/to/myservice",
    "backends": [
        {
            "type": "commandline",
            "commands": {
                "build": "make install",
                "launch": "myservice"
            },
            "launch_checks": {
                "log_text": "Finished startup"
            }
        }
    ]
}
```

## Docker

The Docker backend provides the ability to create and launch Docker containers.
Docker backends have the `type` "docker", and may use an existing image, or build from a Dockerfile.

### Using an existing image

To use an existing image, specify the `image` property using a tag of the form you would use with `docker pull`.

```json
"backends": [
    {
        "type": "docker",
        "image": "kitematic/hello-world-nginx:latest",
        "ports": ["80:8080"]
    }
]
```

The above configuration will create a container from the latest version of the [Kitematic hello world nginx example](https://hub.docker.com/r/kitematic/hello-world-nginx/).
Port 80 on the container will be mapped to port 8080 on the host, so you can browse to the service at http://localhost:8080.

### Building from a Dockerfile

To build a dockerfile, specify the `dockerfile` property as a path to a Dockerfile relative to the service's path.
If the provided path is a directory, the default filename "Dockerfile" will be assumed.

```json
"backends": [
    {
        "type": "docker",
        "dockerfile": "."
    }
]
```

The above example will build and launch a Dockerfile in the service's working directory.

### Additional Configuration

Port mappings are provided as a convenience via the `ports` property.
Additional configuration of a Docker container can be provided using the `containerConfig`, `hostConfig` and `networkConfig` properties with a structure corresponding to the parameters for [ContainerCreate](https://docs.docker.com/engine/api/v1.30/#operation/ContainerCreate) in the Docker Engine API.

```json
"backends": [
    {
        "type": "docker",
        "dockerfile": ".",
        "containerConfig": {...},
        "hostConfig": {...},
        "networkConfig": {...},
    }
]
```

`containerConfig` corresponds to the root paramters of the ContainerCreate command, with `hostConfig` and `networkConfig`
corresponding to `HostConfig` and `NetworkingConfig` respectively.

Field names in each of these objects match those in the Docker Engine API, with the exception that their names are lowerCamelCase in Edward config, instead of the CamelCase of the Docker Engine API.

## Multiple Backends

Sometimes, you may want to use one backend in one situation, and another for a different situation.
For example, you may want to default to building a service locally for most services, but also have the option
of launching the most recent Docker image from your repository.

### Defining multiple backends

You can distinguish between backends using the `name` property.

```json
"backends": [
    {
        "name": "default",
        "type": "commandline",
        ...
    },
    {
        "name": "prebuilt",
        "type": "docker",
        ...
    }
]
```

### Selecting a Backend

By default, Edward will use the first backend in the list for each service.

To start a service with a specific backend, you can use the `--backend`, or `-b` flag at the command line to tell Edward which
backend to use.

    $ edward start -b <service>:<backend> <group>

Will start the service group <group> using the backend <backend> for the service <service>.
Any other services launched as part of the group will use their default backend.