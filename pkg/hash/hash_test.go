package hash

import (
	"fmt"
	"testing"
)

func TestArgon2(t *testing.T) {
	// Пример использования
	password := "mySecurePassword123"

	// Генерация хеша
	encodedHash, err := GenerateHash(password, DefaultArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Хеш:", encodedHash)

	// Проверка пароля
	match, err := VerifyPassword(password, encodedHash)
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Fatal("Пароль не верный (ожидалось true)")
	}
	match, err = VerifyPassword("wrongPassword", encodedHash)
	if err != nil {
		t.Fatal(err)
	}
	if match {
		t.Fatal("Пароль верный (ожидалось false)")
	}
}
