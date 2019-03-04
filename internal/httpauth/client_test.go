package httpauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skywire/pkg/cipher"
)

func TestClient(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == fmt.Sprintf("/security/nonces/%s", pk) {
			json.NewEncoder(w).Encode(&NextNonceResponse{pk, 1}) // nolint: errcheck
		} else {
			require.Equal(t, "1", r.Header.Get("Sw-Nonce"))
			require.Equal(t, pk.Hex(), r.Header.Get("Sw-Public"))
			sig := cipher.Sig{}
			require.NoError(t, sig.UnmarshalText([]byte(r.Header.Get("Sw-Sig"))))
			require.NoError(t, cipher.VerifyPubKeySignedPayload(pk, sig, PayloadWithNonce([]byte{}, 1)))
			fmt.Fprintln(w, "Hello, client")
		}
	}))
	defer ts.Close()

	c, err := NewClient(context.TODO(), ts.URL, pk, sk)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", ts.URL+"/foo", nil)
	require.NoError(t, err)
	res, err := c.Do(req)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, []byte("Hello, client\n"), b)
	assert.Equal(t, uint64(2), c.nonce)
}
