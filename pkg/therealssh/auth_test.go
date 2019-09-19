package therealssh

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/stretchr/testify/require"
)

func TestListAuthorizer(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	auth := &ListAuthorizer{[]cipher.PubKey{pk}}
	require.Error(t, auth.Authorize(cipher.PubKey{}))
	require.NoError(t, auth.Authorize(pk))
}

func TestFileAuthorizer(t *testing.T) {
	pk, _ := cipher.GenerateKeyPair()
	anotherPK, _ := cipher.GenerateKeyPair()

	tmpfile, err := ioutil.TempFile("", "authorized_keys")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.Remove(tmpfile.Name()))
	}()

	_, err = tmpfile.Write([]byte(pk.Hex() + "\n"))
	require.NoError(t, err)

	auth, err := NewFileAuthorizer(tmpfile.Name())
	require.NoError(t, err)

	require.Error(t, auth.Authorize(anotherPK))
	require.NoError(t, auth.Authorize(pk))

	_, err = tmpfile.Write([]byte(anotherPK.Hex() + "\n"))
	require.NoError(t, err)

	require.NoError(t, auth.Authorize(anotherPK))
	require.NoError(t, auth.Authorize(pk))
}
