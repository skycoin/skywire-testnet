package routing

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoltDBRoutingTable(t *testing.T) {
	dbfile, err := ioutil.TempFile("", "routes.db")
	require.NoError(t, err)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		require.NoError(t, os.Remove(dbfile.Name()))
	}()

	tbl, err := BoltDBRoutingTable(dbfile.Name())
	require.NoError(t, err)

	RoutingTableSuite(t, tbl)
}
