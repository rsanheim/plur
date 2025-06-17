## Rux and Docker

I'd like to add the ability to test and exercise Rux inside a Docker container for testing 
on linux architecture, as well as to test file system events on mounted volumes.

As such, we need a docker container that matches the following:

* latest Alpine image
* standard ruby 3.4 installed
* mounted volume for ./rux (so we can build rux from source with the matching architecture)
* mounted volume for ./fixtures/projects (so we have all test projects bidirectionally mounted to the container)
* a small script that we can run on the docker container to build rux, install the watcher binary, and then run the various projects in fixtures/projects
* anything else to make debugging /dev easier on this image as issues arise

## Checklist
