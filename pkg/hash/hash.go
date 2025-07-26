package hash

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strings"
)

type Argon2Params struct {
	Memory      uint32 // Память в KiB (например, 64MB = 65536)
	Iterations  uint32 // Количество итераций
	Parallelism uint8  // Количество потоков
	SaltLength  uint32 // Длина соли (рекомендуется 16)
	KeyLength   uint32 // Длина хеша (рекомендуется 32)
}

var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64MB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

func GenerateHash(password string, params Argon2Params) (string, error) {
	// Генерируем случайную соль
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Вычисляем хеш с помощью Argon2id
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Кодируем хеш и соль в Base64 для хранения
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	// Формат: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	encodedParams := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		encodedSalt,
		encodedHash,
	)

	return encodedParams, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	// Парсим параметры из строки
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	// Вычисляем хеш введенного пароля
	newHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Сравниваем хеши
	if len(hash) != len(newHash) {
		return false, nil
	}

	for i := 0; i < len(hash); i++ {
		if hash[i] != newHash[i] {
			return false, nil
		}
	}

	return true, nil
}

// Парсинг закодированного хеша
func decodeHash(encodedHash string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("неверный формат хеша")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.New("неподдерживаемая версия Argon2")
	}

	params := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, err
	}
	params.SaltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, err
	}
	params.KeyLength = uint32(len(hash))

	return params, salt, hash, nil
}
