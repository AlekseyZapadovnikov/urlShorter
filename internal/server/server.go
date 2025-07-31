package server

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"url-shorter/internal/store"
)

// URLShortener описывает сервис для работы с URL.
type URLShortener interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(alias string) (string, error)
}

type UserService interface {
	RegisterUser(ctx context.Context, mail, hash string) error
	DeleteSession(ctx context.Context, sessionID string) error
	CreateSession(ctx context.Context, userID int64) (string, error)
	GetUserByEmail(ctx context.Context, mail string) (int64, string, error)
	GetUserIDBySessionToken(ctx context.Context, token string) (int64, error)
}

// Server - наш HTTP-сервер.
type Server struct {
	router      *http.ServeMux
	urlService  URLShortener
	userService UserService
	server      *http.Server
}

// New создает и настраивает экземпляр нашего сервера.
func New(addr string, urls URLShortener, usrs UserService) *Server {
	srv := &Server{
		router:      http.NewServeMux(),
		urlService:  urls,
		userService: usrs,
	}
	srv.server = &http.Server{
		Addr:    addr,
		Handler: srv, // Используем srv как обработчик для логирования
	}
	srv.routes() // заполняем router (маршрутизатор)
	return srv
}

// ServeHTTP добавляет логирование ко всем запросам.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("request received", "method", r.Method, "path", r.URL.Path)
	s.router.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// --- Публичные маршруты, доступные всем ---
	s.router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	s.router.HandleFunc("GET /register", s.handleRegisterPage())
	s.router.HandleFunc("POST /register", s.handleRegister())
	s.router.HandleFunc("GET /login", s.handleLoginPage())
	s.router.HandleFunc("POST /login", s.handleLogin())
	s.router.HandleFunc("GET /{alias}", s.handleRedirect()) // Редирект тоже публичный

	// --- Защищенные маршруты, требующие входа ---
	authHandler := http.NewServeMux()
	authHandler.HandleFunc("GET /{$}", s.handleHome()) // Главная страница теперь защищена
	authHandler.HandleFunc("POST /shorten", s.handleShortenURL())
	authHandler.HandleFunc("POST /logout", s.handleLogout()) // Метод POST более корректен для выхода

	// Оборачиваем этот обработчик в middleware и регистрируем на главном роутере
	// Все запросы, начинающиеся с "/", которые не совпали с публичными маршрутами выше,
	// будут направлены сюда и пройдут через проверку аутентификации.
	s.router.Handle("/", s.AuthMiddleware(authHandler))
}

// Start запускает сервер.
func (s *Server) Start() error {
	slog.Info("server starting", "address", s.server.Addr)
	return s.server.ListenAndServe()
}

// ----- Хендлеры для html страниц регистрации и входа -----

var tmpl = template.Must(template.ParseGlob("templates/*.html")) // загрузили все html

func (s *Server) handleRegisterPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "register.html", nil)
	}
}

func (s *Server) handleLoginPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "login.html", nil)
	}
}

// homePage html handler
var homeTmpl = template.Must(template.ParseFiles("templates/home.html"))

type homeData struct {
	ShortURL string
}

func (s *Server) handleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		short := r.URL.Query().Get("short")

		data := homeData{ShortURL: short}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err := homeTmpl.Execute(w, data)
		if err != nil {
			slog.Error("failed to execute template", "error", err)
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

// ----- Хендлеры запросов -----

// запрос на сокращение ссылки
func (s *Server) handleShortenURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "error", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		longURL := r.FormValue("url")
		if longURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		alias, err := s.urlService.CreateShortURL(longURL)
		if err != nil {
			slog.Error("failed to create short url", "error", err)
			http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
			return
		}

		full := "http://" + r.Host + "/" + alias
		// Редиректим на главную страницу с параметром ?short=<полученная ссылка>
		http.Redirect(w, r, "/?short="+url.QueryEscape(full), http.StatusSeeOther)
	}
}

// редирект
func (s *Server) handleRedirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alias := r.PathValue("alias")
		if alias == "" {
			http.NotFound(w, r)
			return
		}

		originalURL, err := s.urlService.GetOriginalURL(alias)
		if err != nil {
			slog.Warn("alias not found", "alias", alias, "error", err)
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusFound)
	}
}

func (s *Server) handleRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mail := r.FormValue("mail")
		pass := r.FormValue("password")

		// TODO: Добавить валидацию пароля
		hash, err := HashPassword(pass)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		err = s.userService.RegisterUser(r.Context(), mail, hash)
		if err != nil {
			if errors.Is(err, store.ErrUserExists) {
				http.Error(w, "User already exists", http.StatusBadRequest)
				return
			}
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		slog.Info("user registered", "mail", mail)
	}
}

func (s *Server) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mail := r.FormValue("mail")
		password := r.FormValue("password")
		slog.Info("начинаем логинить пользователя", "mail", mail, "password", password)

		id, hash, err := s.userService.GetUserByEmail(r.Context(), mail)
		slog.Info("получили пользователя из БД GetUserByEmail", "id", id, "hash", hash, "error", err)
		if err != nil || !CheckPasswordHash(password, hash) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := s.userService.CreateSession(r.Context(), id)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (s *Server) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err == nil {
			s.userService.DeleteSession(r.Context(), cookie.Value)
		}
		clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
