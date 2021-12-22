package prefix

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	ok, partial := Match("foo", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("foo/bar/baz", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bar", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bax", false)
	assert.Equal(t, false, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bar/baz", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("f*", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("foo/bar/*", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/*/baz", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/*/baz", "foo", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/bar/baz", "foo/bar", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/bar/baz", "foo/bax", false)
	assert.Equal(t, false, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/*/baz", "foo/bar/baz", false)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)
}
