### Ruby spec reorg

Our ruby specs have gotten a bit of a mess.  Underneath ./spec we have a mix of things going on:


* specs that test the Plur Ruby housekeeping code for things like installation, release, changelog, etc.
One example of this is the `plur/install_spec.rb` file.
* specs that test the Plur Go CLI from the outside-in, i.e. integration specs for the `plur`, as writing integration specs and scripting style code in Go is a pain.
* probably some other stuff - performance tests? one-off stuff we coudl remove?

Review and analyze the specs, and propose a plan for re-organizing them. The ruby specs should follow standard
ruby conventions (so things like guard or plur (ha!) will work).

Add the plan below in this document.
