package term

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func envOf(vars map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		env        map[string]string
		tty        bool
		wantOn     bool
		wantSource string
	}{
		// Explicit modes short-circuit everything, including contrary env.
		{"always", "always", nil, false, true, "always"},
		{"on alias", "on", nil, false, true, "always"},
		{"never", "never", nil, true, false, "never"},
		{"off alias", "off", nil, true, false, "never"},
		{"always beats NO_COLOR", "always", map[string]string{"NO_COLOR": "1"}, false, true, "always"},
		{"never beats FORCE_COLOR", "never", map[string]string{"FORCE_COLOR": "1"}, true, false, "never"},

		// auto: force vars win first, and are value-sensitive.
		{"auto FORCE_COLOR=1", "auto", map[string]string{"FORCE_COLOR": "1"}, false, true, "FORCE_COLOR"},
		{"auto CLICOLOR_FORCE=1", "auto", map[string]string{"CLICOLOR_FORCE": "1"}, false, true, "CLICOLOR_FORCE"},
		{"auto FORCE_COLOR=0 is not forcing", "auto", map[string]string{"FORCE_COLOR": "0"}, false, false, "not a tty"},
		{"auto FORCE_COLOR=false is not forcing", "auto", map[string]string{"FORCE_COLOR": "false"}, false, false, "not a tty"},
		{"auto FORCE_COLOR empty is not forcing", "auto", map[string]string{"FORCE_COLOR": ""}, false, false, "not a tty"},
		{"auto force beats NO_COLOR", "auto", map[string]string{"FORCE_COLOR": "1", "NO_COLOR": "1"}, false, true, "FORCE_COLOR"},

		// auto: NO_COLOR is presence-based (empty value still counts).
		{"auto NO_COLOR=1", "auto", map[string]string{"NO_COLOR": "1"}, true, false, "NO_COLOR"},
		{"auto NO_COLOR empty", "auto", map[string]string{"NO_COLOR": ""}, true, false, "NO_COLOR"},
		{"auto FORCE_COLOR=0 falls through to NO_COLOR", "auto", map[string]string{"FORCE_COLOR": "0", "NO_COLOR": "1"}, true, false, "NO_COLOR"},

		// auto: tty decides when env is silent.
		{"auto tty", "auto", nil, true, true, "tty"},
		{"auto pipe", "auto", nil, false, false, "not a tty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			on, source := ResolveColor(tc.mode, envOf(tc.env), tc.tty)
			assert.Equal(t, tc.wantOn, on)
			assert.Equal(t, tc.wantSource, source)
		})
	}
}

func TestEnvDecidesColor(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"nothing set", nil, false},
		{"FORCE_COLOR=1", map[string]string{"FORCE_COLOR": "1"}, true},
		{"CLICOLOR_FORCE=1", map[string]string{"CLICOLOR_FORCE": "1"}, true},
		{"NO_COLOR set", map[string]string{"NO_COLOR": ""}, true},
		{"FORCE_COLOR=0 does not decide", map[string]string{"FORCE_COLOR": "0"}, false},
		{"FORCE_COLOR=0 but NO_COLOR set decides", map[string]string{"FORCE_COLOR": "0", "NO_COLOR": "1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, EnvDecidesColor(envOf(tc.env)))
		})
	}
}
