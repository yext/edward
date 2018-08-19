# 2. Implement in Go

Date: 2016-03-02

## Status

Accepted

## Context

Edward will be provided as a command-line tool, ideally across multiple operating systems. It will need a simple means of installation and updating.

## Decision

Edward shall be implemented using Go.

## Consequences

Go applications are cross-platform and can be distributed using `go get`. This means that there is no additional configuration or scripting required. Alternate distribution mechanisms can be built and enabled separately from this repo.

Go also provides built-in support for executing other processes and a basic system for providing command-line interfaces.
