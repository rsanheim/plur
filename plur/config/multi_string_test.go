package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiStringUnmarshalTOML_SingleString(t *testing.T) {
	var ms MultiString
	err := ms.UnmarshalTOML("rspec")

	require.NoError(t, err)
	assert.Equal(t, MultiString{"rspec"}, ms)
}

func TestMultiStringUnmarshalTOML_Array(t *testing.T) {
	var ms MultiString
	err := ms.UnmarshalTOML([]any{"rspec", "minitest"})

	require.NoError(t, err)
	assert.Equal(t, MultiString{"rspec", "minitest"}, ms)
}

func TestMultiStringUnmarshalTOML_InvalidType(t *testing.T) {
	var ms MultiString
	err := ms.UnmarshalTOML(123)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected string or array of strings")
}

func TestMultiStringUnmarshalJSON_SingleString(t *testing.T) {
	var ms MultiString
	data := []byte(`"rspec"`)

	err := json.Unmarshal(data, &ms)

	require.NoError(t, err)
	assert.Equal(t, MultiString{"rspec"}, ms)
}

func TestMultiStringUnmarshalJSON_Array(t *testing.T) {
	var ms MultiString
	data := []byte(`["rspec", "minitest"]`)

	err := json.Unmarshal(data, &ms)

	require.NoError(t, err)
	assert.Equal(t, MultiString{"rspec", "minitest"}, ms)
}

func TestMultiStringSlice(t *testing.T) {
	ms := MultiString{"rspec", "minitest"}
	copy := ms.Slice()

	assert.Equal(t, []string{"rspec", "minitest"}, copy)

	// Verify it's a copy by modifying original
	ms[0] = "changed"
	assert.Equal(t, "rspec", copy[0], "Slice() should return a copy, not the original")
}
