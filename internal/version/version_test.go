package version

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShort_ReturnsVersion(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()

	Version = "1.2.3"
	assert.Equal(t, "1.2.3", Short())
}

func TestFull_ShortCommitHash(t *testing.T) {
	origV, origC := Version, Commit
	defer func() { Version, Commit = origV, origC }()

	Version = "0.5.0"
	Commit = "abc1234def890567"

	result := Full()
	assert.Equal(t, "0.5.0 (abc1234)", result)
}

func TestFull_ShortCommit(t *testing.T) {
	origV, origC := Version, Commit
	defer func() { Version, Commit = origV, origC }()

	Version = "0.1.0"
	Commit = "abc"

	result := Full()
	assert.Equal(t, "0.1.0 (abc)", result)
}

func TestFull_ExactlySevenChars(t *testing.T) {
	origV, origC := Version, Commit
	defer func() { Version, Commit = origV, origC }()

	Version = "1.0.0"
	Commit = "abcdefg"

	result := Full()
	assert.Equal(t, "1.0.0 (abcdefg)", result)
}

func TestPlatform_Format(t *testing.T) {
	expected := runtime.GOOS + "/" + runtime.GOARCH
	assert.Equal(t, expected, Platform())
}

func TestDetailed_ContainsAllFields(t *testing.T) {
	origV, origC, origD, origG := Version, Commit, Date, GoVersion
	defer func() { Version, Commit, Date, GoVersion = origV, origC, origD, origG }()

	Version = "2.0.0"
	Commit = "deadbeef"
	Date = "2026-01-15T10:00:00Z"
	GoVersion = "go1.25.0"

	result := Detailed()
	assert.Contains(t, result, "ssherpa 2.0.0")
	assert.Contains(t, result, "Commit:    deadbeef")
	assert.Contains(t, result, "Built:     2026-01-15T10:00:00Z")
	assert.Contains(t, result, "Go:        go1.25.0")
	assert.Contains(t, result, "Platform:  "+runtime.GOOS+"/"+runtime.GOARCH)
}
