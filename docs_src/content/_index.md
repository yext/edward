---
date: 2016-03-08T21:07:13+01:00
title: About Edward
type: index
---

A command line tool for managing local instances of microservices.

Edward will build and launch groups of services in your development environment
with just a single call to `edward start`. No more long rows of Terminal tabs!

Services run in the background, and you can follow the logs for a single service
as and when you need with `edward log`

Need to rebuild? Just run `edward restart`, or set up a watch on the source folder.

When you're all done for the day, just run `edward stop`.

## Features

* Group services with a single alias
* Prompt for a password when sudo is required
* Auto-generate service configuration for:
  * Go
  * Docker
  * ICBM
* Watch directories for automatic rebuild
* "Warm up" services by sending an HTTP request

## Acknowledgements
