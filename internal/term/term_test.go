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
		{"true alias", "true", nil, false, true, "always"},
		{"never", "never", nil, true, false, "never"},
		{"false alias", "false", nil, true, false, "never"},
		{"always beats NO_COLOR", "always", map[string]string{"NO_COLOR": "1"}, false, true, "always"},

		// auto: NO_COLOR is presence-based (empty value still counts).
		{"auto NO_COLOR=1", "auto", map[string]string{"NO_COLOR": "1"}, true, false, "NO_COLOR"},
		{"auto NO_COLOR empty", "auto", map[string]string{"NO_COLOR": ""}, true, false, "NO_COLOR"},

		// auto: tty decides when NO_COLOR is absent.
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
		{"NO_COLOR set (empty)", map[string]string{"NO_COLOR": ""}, true},
		{"NO_COLOR=1", map[string]string{"NO_COLOR": "1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, EnvDecidesColor(envOf(tc.env)))
		})
	}
}
