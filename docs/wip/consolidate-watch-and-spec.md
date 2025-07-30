# Research opportunities for consolidating 'plur spec' and 'plur watch'

## Problem

Currently, 'plur spec' and 'plur watch' use distinct designs and code paths for most of how they work. One runs tests in paralle, while the other uses file based events to run specs in response. This will make shared behavior where it makes sense difficult.

Lets research and analyze the following:

* what _is_ shared between the two commands?
* what is not shared between the two commands?
* where does it make sense to refactor to consolidate to the same code paths ?
* where does it NOT make sense to refactor to consolidate to the same code paths ?

## Opportunities

* Allowing plur watch to run tests in parallel would be much easier if it shared the worker/stream based design of plur spec
* They both support rspec and minitest. In the future, we'd like to support other test frameworks more generally, not just ruby. 
* We could allow formatting changes or tweaks to apply consistently across both commands
* (maybe) shared 'TUI" type commands that work the same between watch and spec, i.e. ability to show existing config, or change it on the fly while in watch
* Easier maintenance long term

## Challenges

## Findings

_assistant notes go here_