package ed25519consensus

import (
	"crypto/ed25519"
	"testing"

	"github.com/tendermint/tendermint/crypto"
)

func TestBatch(t *testing.T) {
	// make a bunch of keys
	// sign a distince message for each key
	// create keysigs add them to an array of them
	// call batch
	v := NewVerifier()
	for i := 0; i <= 2; i++ {

		pub, priv, _ := ed25519.GenerateKey(crypto.CReader())

		msg := []byte("BatchVerifyTest")

		sig := ed25519.Sign(priv, msg)

		if !Verify(pub, msg, sig) {
			t.Error("failure to verify single key")
		}

		if !v.Add(pub, sig, msg) {
			t.Error("unable to add s k m")
		}
	}

	if !v.BatchVerify() {
		t.Error("failed batch verification")
	}
}
