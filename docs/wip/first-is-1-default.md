For cases where plur auto sets TEST_ENV_NUMBER, we want to auto set the _first_ worker to '1' -- this is to ensure that the default DB remains unused during parallel runs.  This should be controlled via a flag '--first-is-1' which will default to true.

For example, this means in the default case, if someone runs `plur spec -n4`, workers will run:

worker 0 -> TEST_ENV_NUMBER=1 rspec [files]
worker 1 -> TEST_ENV_NUMBER=2 rspec [files]
worker 2 -> TEST_ENV_NUMBER=3 rspec [files]
worker 3 -> TEST_ENV_NUMBER=4 rspec [files]

If someone runs `plur spec -n4 --first-is-1=false`, it should work as it does right now, where the first worker gets '', and the rest get incremented by 1.

worker 0 -> TEST_ENV_NUMBER='' rspec [files]
worker 1 -> TEST_ENV_NUMBER=2 rspec [files]
worker 2 -> TEST_ENV_NUMBER=3 rspec [files]
worker 3 -> TEST_ENV_NUMBER=4 rspec [files]

This means plur will default to what is the sane, predictable behavior of using explicit numbered DBs for parallel runs.

Note that if someone is running in 'serial' mode, i.e. --n=1, then we should not set the TEST_ENV_NUMBER at all.

### Checklist
