| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `rux -n 4 --command="bundle exec rspec"` | 9.130 ± 0.038 | 9.090 | 9.164 | 1.00 |
| `rux -n 4 --command="bin/rspec"` | 9.145 ± 0.045 | 9.100 | 9.190 | 1.00 ± 0.01 |

## Summary

**bin/rspec is -0.2% faster than bundle exec rspec**
