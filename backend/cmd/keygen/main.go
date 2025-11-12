package main

import (
	"hackathon-back/pkg/jwt"
)

func main() {
	jwt.MustECDSAGenerateKeys()
}
