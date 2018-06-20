---
date: 2017-01-15T22:16:39-05:00
title: Troubleshooting
---

## Services can't be started or stopped after upgrading

After upgrading Edward, if you experience problems stopping or starting services, or you appear to have "orphaned" services running in the background, this may be a result of corrupted state files.

Steps to reset state:

1. Restart your computer
2. Ensure there are no services running under Edward
3. Delete the Edward home directory: `rm ~/.edward`

This will completely reset Edward to a clean state.
