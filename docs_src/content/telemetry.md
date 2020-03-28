---
date: 2017-01-15T22:16:39-05:00
title: Telemetry
---

Edward allows you to specify a script to trigger telemetry events with the telemetry
provider of your choice.

Specify your script at the root of your *edward.json* file:

```json
{
  "telemetryScript": "myScript.sh",
  "imports": ...,
  "groups": ...,
  "services": ...
}
```

Currently, this script is run in the background whenever `edward start` is run. The script will be run with parameters of the form `<command> [services/groups]`.

So if you run against the above *edward.json* with the command `edward start group1 service1`, the following command will be run `myScript.sh start group1 service1`.

Note that this script runs in the background, so if `edward start` completes particularly quickly, it may not run to completion.
