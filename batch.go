package ed25519consensus

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"filippo.io/edwards25519"
)

var B = edwards25519.NewGeneratorPoint()

type Verifier struct {
	signatures []ks
	batchSize  uint32
}

type ks struct {
	pubkey     ed25519.PublicKey
	signatures []sm
}

type sm struct {
	signature signature
	msg       []byte
	k         *edwards25519.Scalar
}

type signature struct {
	rBytes [32]byte // 0..32
	sBytes [32]byte // 32..64
}

func NewVerifier() Verifier {
	return Verifier{
		signatures: []ks{},
		batchSize:  0,
	}
}

func (v *Verifier) Add(publicKey ed25519.PublicKey, sig, message []byte) bool {
	if l := len(publicKey); l != ed25519.PublicKeySize {
		return false
	}

	if len(sig) != ed25519.SignatureSize || sig[63]&224 != 0 {
		return false
	}

	var (
		rBytes [32]byte
		sBytes [32]byte
	)
	copy(rBytes[:], sig[:32])
	copy(sBytes[:], sig[32:64])

	s := signature{
		rBytes: rBytes,
		sBytes: sBytes,
	}

	smS := sm{
		signature: s,
		msg:       message,
	}

	ksS := ks{
		pubkey:     pk,
		signatures: []sm{smS},
	}

	v.batchSize++
	v.signatures[int(v.batchSize)] = ksS

	return true
}

func (v *Verifier) BatchVerify() bool {
	// The batch verification equation is
	//
	// [-sum(z_i * s_i)]B + sum([z_i]R_i) + sum([z_i * k_i]A_i) = 0.
	// where for each signature i,
	// - A_i is the verification key;
	// - R_i is the signature's R value;
	// - s_i is the signature's s value;
	// - k_i is the hash of the message and other data;
	// - z_i is a random 128-bit Scalar.

	var (
		A_coeffs = make([]*edwards25519.Scalar, 0, v.batchSize)
		R_coeffs []*edwards25519.Scalar

		As = make([]*edwards25519.Point, 0, v.batchSize)
		Rs []*edwards25519.Point
	)

	B_coeff := edwards25519.NewScalar()

	for i := 0; i < int(v.batchSize); i++ {
		A, ok := Decompress(v.signatures[i].pubkey)
		if !ok {
			return false
		}

		A_coeff := edwards25519.NewScalar()

		for j := 0; j < len(v.signatures[i].signatures); j++ {

			s, err := new(edwards25519.Scalar).SetCanonicalBytes(v.signatures[i].signatures[j].signature[32:])
			if err != nil {
				return false
			}

			buf := make([]byte, 32)
			_, _ = rand.Read(buf[:16])
			z, err := new(edwards25519.Scalar).SetCanonicalBytes(buf)
			if err != nil {
				return false
			}

			b, ok := Decompress(v.signatures[i].signatures[j].signature.rBytes[:])
			if !ok {
				return false
			}

			Rs = append(Rs, b)
			R_coeffs = append(R_coeffs, z)

			z.Multiply(z, v.signatures[i].signatures[j].k)
			A_coeff.Add(z, A_coeff)
		}

		As = append(As, A)
		A_coeffs = append(A_coeffs, A_coeff)
	}

	var (
		scalars []*edwards25519.Scalar
		points  []*edwards25519.Point
	)

	points = append(points, B)
	points = append(points, As...)
	points = append(points, Rs...)

	scalars = append(scalars, B_coeff)
	scalars = append(scalars, A_coeffs...)
	scalars = append(scalars, R_coeffs...)

	check := new(edwards25519.Point).VarTimeMultiScalarMult(scalars, points)
	check.MultByCofactor(check)

	fmt.Println(check, "check", edwards25519.NewIdentityPoint())
	if check.Equal(edwards25519.NewIdentityPoint()) == 1 {
		return true
	} else {
		return false
	}
}
