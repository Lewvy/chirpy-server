package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type ArgonConfig struct {
	HashRaw   []byte
	Salt      []byte
	TimeCost  uint32
	MemCost   uint32
	Threads   uint8
	KeyLength uint32
}

// Create a hashed password
func generateSalt(saltSize uint32) ([]byte, error) {
	salt := make([]byte, saltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("salt generation failed: %w", err)
	}
	return salt, nil
}

func HashPassword(pwd string) (*string, error) {
	config := &ArgonConfig{
		TimeCost:  2,
		MemCost:   64 * 1024,
		Threads:   4,
		KeyLength: 32,
	}

	salt, err := generateSalt(16)
	if err != nil {
		return nil, fmt.Errorf("Error generating salt: %w", err)
	}
	config.HashRaw = argon2.IDKey([]byte(pwd), salt, config.TimeCost, config.MemCost, config.Threads, config.KeyLength)
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		config.MemCost,
		config.TimeCost,
		config.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(config.HashRaw),
	)
	return &encodedHash, nil
}

func parseArgon2Hash(storedHash string) (*ArgonConfig, error) {
	components := strings.Split(storedHash, "$")
	if len(components) != 6 {
		return nil, errors.New("invalid hash format structure")
	}

	if !strings.HasPrefix(components[1], "argon2id") {
		return nil, errors.New("unsupported algorithm variant")
	}

	var version int
	_, err := fmt.Sscanf(components[2], "v=%d", &version)
	if err != nil {
		return nil, errors.New("error reading version")
	}

	config := &ArgonConfig{}
	_, err = fmt.Sscanf(components[3], "m=%d,t=%d,p=%d", &config.MemCost, &config.TimeCost, &config.Threads)
	if err != nil {
		return nil, errors.New("error reading components")
	}
	salt, err := base64.RawStdEncoding.DecodeString(components[4])
	if err != nil {
		return nil, fmt.Errorf("salt decoding failed: %w", err)
	}
	config.Salt = salt

	hash, err := base64.RawStdEncoding.DecodeString(components[5])
	if err != nil {
		return nil, fmt.Errorf("hash decoding failed: %w", err)
	}

	config.HashRaw = hash
	config.KeyLength = uint32(len(hash))
	return config, nil
}

func VerifyHashedPw(storedHash, providedPwd string) (bool, error) {
	config, err := parseArgon2Hash(storedHash)
	if err != nil {
		return false, fmt.Errorf("Hash parsing failed: %w", err)
	}

	computedHash := argon2.IDKey(
		[]byte(providedPwd),
		config.Salt,
		config.TimeCost,
		config.MemCost,
		config.Threads,
		config.KeyLength,
	)

	match := subtle.ConstantTimeCompare(config.HashRaw, computedHash) == 1
	return match, nil
}
