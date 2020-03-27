package main

import (
	"crypto/sha256"
	"encoding/base64"
)

//MARK: Constants
const (
	GoogleDriveClientSecret = "credentials.json"
)

//MARK: Helper Functions

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// MARK: SHA256 Helpers

func SHA256Base64URL(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func checkSHA2(hash string, data []byte) bool {
	return SHA256Base64URL(data) == hash
}
