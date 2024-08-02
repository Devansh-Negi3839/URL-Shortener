package main

import (
	"crypto/sha256"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Base62Encode encodes a byte slice to a Base62 string.
func base62Encode(bytes []byte) string {
	// Convert byte slice to a big integer
	num := new(big.Int).SetBytes(bytes)

	// Encode the big integer to Base62
	var encoded string
	base := big.NewInt(62)
	zero := big.NewInt(0)
	for num.Cmp(zero) > 0 {
		mod := new(big.Int)
		num.DivMod(num, base, mod)
		encoded = string(base62Chars[mod.Int64()]) + encoded
	}

	return encoded
}

// hashUrl hashes a URL using SHA-256 and returns an 8-character Base62 encoded string.
func hashUrl(url string) string {
	hash := sha256.New()
	hash.Write([]byte(url))
	hashBytes := hash.Sum(nil)

	// Encode to Base62 and truncate to 8 characters
	base62Encoded := base62Encode(hashBytes)
	if len(base62Encoded) > 8 {
		return base62Encoded[:8]
	}
	return base62Encoded
}