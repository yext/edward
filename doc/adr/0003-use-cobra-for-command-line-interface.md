# 3. Use Cobra for command line interface

Date: 2017-08-12

## Status

Accepted

## Context

Edward provides a rich command-line interface intended to use a command structure familiar to most developers.

## Decision

[Cobra](https://github.com/spf13/cobra) will be used as a CLI framework.

## Consequences

Cobra provides an opinionated structure for command-line applications, allowing functionality to be separated from the user interface. This will make adding new commands and unit testing functional code easier in future.
