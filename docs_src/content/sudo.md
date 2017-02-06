---
date: 2017-01-15T22:16:39-05:00
title: Sudo
---

Edward will not run if you try to launch it with sudo, but it may ask you to provide your password so that certain services can be run with elevated priviledges. The password request is triggered through a bash script that calls a command with sudo, to ensure that your bash session can make further sudo calls without prompting.

This has only been tested in one bash environment, so your mileage may vary. If services hang when starting (waiting for their log), this may be an indicator that they are waiting for a password prompt that isn't redirected anywhere.
