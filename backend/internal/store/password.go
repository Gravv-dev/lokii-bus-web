package store

import "golang.org/x/crypto/bcrypt"

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func passwordMatches(hash, password []byte) bool {
	return bcrypt.CompareHashAndPassword(hash, password) == nil
}
