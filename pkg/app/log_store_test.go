package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogStore(t *testing.T) {
	p, err := ioutil.TempFile("", "test-db")
	require.NoError(t, err)

	defer os.Remove(p.Name()) // nolint

	ls, err := newBoltDB(p.Name(), "foo")
	require.NoError(t, err)

	t3, err := time.Parse(time.RFC3339, "2000-03-01T00:00:00Z")
	require.NoError(t, err)

	err = ls.Store(t3, "foo")
	require.NoError(t, err)

	t1, err := time.Parse(time.RFC3339, "2000-01-01T00:00:00Z")
	require.NoError(t, err)

	err = ls.Store(t1, "bar")
	require.NoError(t, err)

	t2, err := time.Parse(time.RFC3339, "2000-02-01T00:00:00Z")
	require.NoError(t, err)

	err = ls.Store(t2, "middle")
	require.NoError(t, err)

	res, err := ls.LogsSince(t1)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Contains(t, res[0], "middle")
	require.Contains(t, res[1], "foo")

	t4, err := time.Parse(time.RFC3339, "1999-02-01T00:00:00Z")
	require.NoError(t, err)
	res, err = ls.LogsSince(t4)
	require.NoError(t, err)
	require.Len(t, res, 3)
	require.Contains(t, res[0], "bar")
	fmt.Println("b_ :", res[0])
	require.Contains(t, res[1], "middle")
	require.Contains(t, res[2], "foo")
}
