# Edward Prototype

This project is a prototype of the Edward local service management tool.

Full proposal document at: https://docs.google.com/a/yext.com/document/d/16Y8qWTlYDmuEnJgYKTcKb98a2Te-xqdD11auOJ7fUtM/edit?usp=sharing

## Installation

    go get stash.office.yext.com/scm/~telliott/edward-prototype.git
    
This will download and install the tool as 'edward-prototype.git' in your go bin directory. The .git suffix is an unfortunate side effect of using Stash, but there is no harm in renaming the executable if you wish.

## Usage

    NAME:
       Edward - Manage local microservices
    
    USAGE:
       edward-prototype.git [global options] command [command options] [arguments...]
    
    VERSION:
       0.0.0

    COMMANDS:
        list	    List available services
        generate	 Generate Edward config for a source tree
        status	 Display service status
        messages	 Show messages from services
        start   	 Build and launch a service
        stop	    Stop a service
        restart	 Rebuild and relaunch a service
        log		 Tail the log for a service

    GLOBAL OPTIONS:
       --help, -h		show help
       --version, -v	print the version

At the time of writing, the generate and messages commands are not implemented.

## Running Pages

This prototype includes hard-coded configuration for third party tools, a basic Storm setup, and Pages.

The *pages* group has been provided to collect all the services that make up Pages, including *sites-staging*, *sites-cog*, *sites-storm* and other services on which they depend.

To start the services necessary for running Pages locally:

    edward-prototype.git start pages
    
This will build each service, before starting them in sequence. Failure in any build process will stop the command and nothing will launch. Failure in a start will stop further progress, but will not stop already running services
    
Once they are running, you can stop them with the command:

    edward-prototype.git stop pages
    
If you want to view the logs for this run of sites-cog, for example, you can call:

    edward-prototype.git log sites-cog
    
Note that you can only do this for a single service, `log pages`, for example, will cause an error.

## sudo

Edward will not run if you try to launch it with sudo, but it may ask you to provide your password so that certain services (namely rabbitmq and haproxy) can be run with elevated priviledges. The password request is triggered through a bash script that calls a command with sudo, to ensure that your bash session can make further sudo calls without prompting.

This has only been tested in one bash environment, so your mileage may vary. If rabbitmq or haproxy hang when starting (waiting for their log), this may be an indicator that they are waiting for a password prompt that isn't redirected anywhere.