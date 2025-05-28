# Rspec Package

## We have a lot of mixed concerns throughout the Runner and other files.

Lets break out an `rspec` that contains responsibilities for running and processing rspec files.
This will help us divide up responsibilities, refine interfaces, and paves the way for perf work 
as well as adding the abililtiy to handle minitest and other test runners down the road.

Do not prematurely generalize, but lets start refactoring towards something cleaner.

Questions:
* any duplication we should clean up first? 
* Or interfaces to extract?
* Is RuntimeTracker general enough for other test runners? Or assume its rspec focused for now?

## Proposed Structure / type break out

TODO
