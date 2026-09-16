package term

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveFormatter(t *testing.T) {
	cases := []struct {
		name      string
		formatter Formatter
		tty       bool
		want      Formatter
	}{
		{name: "auto tty", formatter: FormatterAuto, tty: true, want: FormatterProgress},
		{name: "auto pipe", formatter: FormatterAuto, want: FormatterSummary},
		{name: "progress tty", formatter: FormatterProgress, tty: true, want: FormatterProgress},
		{name: "progress pipe", formatter: FormatterProgress, want: FormatterProgress},
		{name: "summary tty", formatter: FormatterSummary, tty: true, want: FormatterSummary},
		{name: "summary pipe", formatter: FormatterSummary, want: FormatterSummary},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveFormatter(tc.formatter, tc.tty))
		})
	}
}

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		noColorSet bool
		noColorVal string
		tty        bool
		wantOn     bool
		wantSource string
	}{
		// Explicit modes short-circuit everything, including contrary env.
		{name: "always", mode: "always", wantOn: true, wantSource: "always"},
		{name: "true alias", mode: "true", wantOn: true, wantSource: "always"},
		{name: "never", mode: "never", tty: true, wantSource: "never"},
		{name: "false alias", mode: "false", tty: true, wantSource: "never"},
		{name: "always beats NO_COLOR", mode: "always", noColorSet: true, noColorVal: "1", wantOn: true, wantSource: "always"},

		// auto: NO_COLOR is presence-based (empty value still counts).
		{name: "auto NO_COLOR=1", mode: "auto", noColorSet: true, noColorVal: "1", tty: true, wantSource: "NO_COLOR"},
		{name: "auto NO_COLOR empty", mode: "auto", noColorSet: true, tty: true, wantSource: "NO_COLOR"},

		// auto: tty decides when NO_COLOR is absent.
		{name: "auto tty", mode: "auto", tty: true, wantOn: true, wantSource: "tty"},
		{name: "auto pipe", mode: "auto", wantSource: "not a tty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tc.noColorVal) // registers cleanup; also isolates from the outer env
			if !tc.noColorSet {
				os.Unsetenv("NO_COLOR")
			}
			on, source := ResolveColor(tc.mode, tc.tty)
			assert.Equal(t, tc.wantOn, on)
			assert.Equal(t, tc.wantSource, source)
		})
	}
}
