package server

import (
	"context"
	"net/http"
)

// Определим ключ для контекста, чтобы избежать коллизий
type contextKey string
const userContextKey = contextKey("userID")

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем куки
		cookie, err := r.Cookie("session_token")
		if err != nil {
			// Если куки нет, отправляем на страницу входа
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Проверяем токен сессии в БД
		token := cookie.Value
		userID, err := s.userService.GetUserIDBySessionToken(r.Context(), token)
		if err != nil {
			// Если сессия не найдена или истекла, удаляем куки и отправляем на вход
			clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Сохраняем ID пользователя в контексте запроса
		// Это позволит другим хендлерам знать, какой пользователь отправил запрос
		ctx := context.WithValue(r.Context(), userContextKey, userID)

		// Вызываем следующий обработчик в цепочке с обновленным контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Хелпер для получения userID из контекста в других хендлерах
func getUserIDFromContext(ctx context.Context) (int, bool) {
    userID, ok := ctx.Value(userContextKey).(int)
    return userID, ok
}