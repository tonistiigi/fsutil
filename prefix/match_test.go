package prefix

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	ok, partial := Match("foo", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("foo/bar/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bax")
	assert.Equal(t, false, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/bar/baz", "foo/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("f*", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)

	ok, partial = Match("foo/bar/*", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("foo/*/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/*/baz", "foo")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/bar/baz", "foo/bar")
	assert.Equal(t, true, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/bar/baz", "foo/bax")
	assert.Equal(t, false, ok)
	assert.Equal(t, true, partial)

	ok, partial = Match("*/*/baz", "foo/bar/baz")
	assert.Equal(t, true, ok)
	assert.Equal(t, false, partial)
}
