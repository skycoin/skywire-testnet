package httpauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/SkycoinProject/dmsg/cipher"
	"github.com/SkycoinProject/skycoin/src/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	payload      = "Hello, client\n"
	errorMessage = `{"error":{"message":"SW-Nonce does not match","code":401}}`
)

func TestMain(m *testing.M) {
	loggingLevel, ok := os.LookupEnv("TEST_LOGGING_LEVEL")
	if ok {
		lvl, err := logging.LevelFromString(loggingLevel)
		if err != nil {
			log.Fatal(err)
		}
		logging.SetLevel(lvl)
	} else {
		logging.Disable()
	}

	os.Exit(m.Run())
}

func TestClient(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	headerCh := make(chan http.Header, 1)
	ts := newTestServer(t, pk, headerCh)
	defer ts.Close()

	c, err := NewClient(context.TODO(), ts.URL, pk, sk)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", ts.URL+"/foo", bytes.NewBufferString(payload))
	require.NoError(t, err)
	res, err := c.Do(req)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, []byte(payload), b)
	assert.Equal(t, uint64(2), c.nonce)

	headers := <-headerCh
	checkResp(t, headers, b, pk, 1)
}

// TestClient_BadNonce tests if `Client` retries the request if an invalid nonce is set.
func TestClient_BadNonce(t *testing.T) {
	pk, sk := cipher.GenerateKeyPair()

	headerCh := make(chan http.Header, 1)
	ts := newTestServer(t, pk, headerCh)
	defer ts.Close()

	c, err := NewClient(context.TODO(), ts.URL, pk, sk)
	require.NoError(t, err)

	c.nonce = 999

	req, err := http.NewRequest("GET", ts.URL+"/foo", bytes.NewBufferString(payload))
	require.NoError(t, err)
	res, err := c.Do(req)
	require.NoError(t, err)

	b, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	assert.Equal(t, uint64(2), c.nonce)

	headers := <-headerCh
	checkResp(t, headers, b, pk, 1)
}

func checkResp(t *testing.T, headers http.Header, body []byte, pk cipher.PubKey, nonce int) {
	require.Equal(t, strconv.Itoa(nonce), headers.Get("Sw-Nonce"))
	require.Equal(t, pk.Hex(), headers.Get("Sw-Public"))
	sig := cipher.Sig{}
	require.NoError(t, sig.UnmarshalText([]byte(headers.Get("Sw-Sig"))))
	require.NoError(t, cipher.VerifyPubKeySignedPayload(pk, sig, PayloadWithNonce(body, Nonce(nonce))))
}

func newTestServer(t *testing.T, pk cipher.PubKey, headerCh chan<- http.Header) *httptest.Server {
	nonce := 1
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == fmt.Sprintf("/security/nonces/%s", pk) {
			require.NoError(t, json.NewEncoder(w).Encode(&NextNonceResponse{pk, Nonce(nonce)}))
		} else {
			body, err := ioutil.ReadAll(r.Body)
			if body != nil {
				defer func() {
					require.NoError(t, r.Body.Close())
				}()
			}
			if err != nil {
				return
			}
			respMessage := string(body)
			if r.Header.Get("Sw-Nonce") != strconv.Itoa(nonce) {
				respMessage = errorMessage
			} else {
				headerCh <- r.Header
				nonce++
			}
			_, err = fmt.Fprint(w, respMessage)
			require.NoError(t, err)
		}
	}))
}
