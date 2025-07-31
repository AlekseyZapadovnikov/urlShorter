// возможно в дальнейшем это стоит вынести в отдельный пакет auth
// или убрать в пакет service или service/auth, но пусть пока лежит тут
package server

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	hashCost = 10
	// стоимость хеширования, чем больше, тем сложнее хеш, но и дольше он будет генерироваться
	// сложность 2**n => 2^10 = 1024 итераций будет генерироваться хэш для пароля
)

// Хелпер для хеширования пароля
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	return string(bytes), err
}

// Хелпер для проверки пароля
func CheckPasswordHash(password, hash string) bool {
	fmt.Println("hash", hash, "password", password)
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Хелперы для работы с куки
func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})
}
