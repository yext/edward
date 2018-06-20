---
date: 2017-01-15T22:29:28-05:00
title: Autocompletion
---

To enable bash autocompletion, create a file with the following:

    #! /bin/bash

    : ${PROG:=$(basename ${BASH_SOURCE})}

    _cli_bash_autocomplete() {
        local cur opts base
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        opts=$( ${COMP_WORDS[@]:0:$COMP_CWORD} --generate-bash-completion )
        local w matches=()
        local nocasematchWasOff=0
        shopt nocasematch >/dev/null || nocasematchWasOff=1
        (( nocasematchWasOff )) && shopt -s nocasematch
        for w in $opts; do
            if [[ "$w" == "$cur"* ]]; then matches+=("$w"); fi
        done
        (( nocasematchWasOff )) && shopt -u nocasematch
        COMPREPLY=("${matches[@]}")
        return 0
    }

    complete -F _cli_bash_autocomplete $PROG

Then source it from your bash profile

    PROG=edward source FILE

Alternatively, name the file edward and place it in your system appropriate `bash_completion.d/` directory.
