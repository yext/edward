---
date: 2017-01-15T22:16:39-05:00
title: Troubleshooting
---

## Services can't be started or stopped after upgrading

After upgrading Edward, if you experience problems stopping or starting services, this may be a result of corrupted state files.

Steps to reset state:

1. Restart your computer
2. Ensure there are no services running under Edward
3. Delete the Edward home directory: `rm ~/.edward`

This will completely reset Edward to a clean state.

## Orphaned services running after upgrade or crash

After an upgrade or if Edward exits unexpectedly while starting/stopping services, you may find that you have orphaned services running in the background.

You can terminate all Edward related processes using pkill:

`$ pkill -f edward`

This will find and kill all services with 'edward' in their command.