# Interactive Plur

One of my 'big idea' goals with Plur is to allow a developer to use `plur watch` to build up an interactive
config and mapping of files to tests while they are running.  So here is one simple example:

User starts `plur watch` and saves the file `app/services/user_publisher.rb`

plur does not find a direct mapping (as it would be spec/services/user_publisher_spec.rb only with our simple mapping rules), and tells the user that, but in presents the user with options of other specs that could be run:

* spec/services/user_publisher_spec.rb
* spec/integrations/user_publisher_spec.rb
* spec/*user_publisher**.rb

and so on.  

The basic idea is to allow a developer to add rules to their config interactively based on files saved that 
_do not_ have any matching files to run....and there are some pretty easy heuristics we can apply to handle
90% of cases for this sort of thing. I think. 

Some contraints:
* we should not try to be too clever -- providing a general glob rule based on a file saved is a good start
* if a user saves 'app/models/user.rb', and there is a typiocal matching spec 'spec/models/user_spec.rb', we should not try to provide suggestions
* if we provide suggestions, we should tell the user what specs _would_ match if they added that rule to make the mathcing work
* we should provide enough feedback to the user to help them understand how mapping rules work with plur, and how they can tweak them later to get more specific or correct

Additionally, we should allow watch to be in two different modes that can be toggled by the user:
* 'learn' mode: where plur will suggest rules to add to the config based on the files saved
* 'standard' mode: where plur will run just what is in the config file as prescribed

by default `plur watch` will be in the standard mode, but I think the learn mode could be an attractive offering for more complicated test suites...and help us build out our mapping rules to suit the many varieties of test - to lib spec rules.

### Implementation Plan

* Consider how the config is loaded and how we can change it at runtime (for live feedback), and also write it back to the file system to save valid rules for the user
* Consider how to make this user friendly: we want to explain what the rules currently are (and why), and then explain what plur is suggesting, and then explain changes plur may make to the runtime rules and the config saved on disk.
* a broader goal is to help developers think thru what test files are important when a certain file changes....and providing input and guidance to help them build the correct set of matching rules that respond to file change events. This may mean running specs that match a simple glob pattern, or if someone saves "config/application.rb", we can suggest just running the entire suite or maybe running "spec/config/application_spec.rb" if it exists.  

