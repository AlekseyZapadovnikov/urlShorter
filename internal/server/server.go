package server

import (
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
)

// URLShortener описывает сервис для работы с URL.
type URLShortener interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(alias string) (string, error)
}

// Server - наш HTTP-сервер.
type Server struct {
	router  *http.ServeMux
	service URLShortener
	server  *http.Server
}

// New создает и настраивает экземпляр нашего сервера.
func New(addr string, s URLShortener) *Server {
	srv := &Server{
		router:  http.NewServeMux(),
		service: s,
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

// routes регистрирует пути. Порядок важен!
func (s *Server) routes() {
	// 1. Главная страница, которая показывает форму (только GET)
	s.router.HandleFunc("GET /", s.handleHome())

	// 2. Обработчик для создания короткой ссылки (только POST)
	s.router.HandleFunc("POST /shorten", s.handleShortenURL())

	// 3. Обработчик для редиректа. Он должен быть последним,
	// так как он является "всеядным" (wildcard).
	s.router.HandleFunc("GET /{alias}", s.handleRedirect())
}

// Start запускает сервер.
func (s *Server) Start() error {
	slog.Info("server starting", "address", s.server.Addr)
	return s.server.ListenAndServe()
}

// ----- Хендлеры -----

func (s *Server) handleShortenURL() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Читаем данные из формы
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

		// TODO: Пока передаем пустой email. В будущем здесь будет логика аутентификации.
		alias, err := s.service.CreateShortURL(longURL)
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

func (s *Server) handleRedirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		alias := r.PathValue("alias")
		if alias == "" {
			http.NotFound(w, r)
			return
		}

		originalURL, err := s.service.GetOriginalURL(alias)
		if err != nil {
			// Здесь можно проверить тип ошибки: если "не найдено", то 404
			slog.Warn("alias not found", "alias", alias, "error", err)
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, originalURL, http.StatusFound)
	}
}

// homePage handler
var homeTmpl = template.Must(template.ParseFiles("templates/home.html"))

type homeData struct {
	ShortURL string
}

func (s *Server) handleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Если это не GET / , то это 404.
		// Это дополнительная проверка, т.к. роутер уже должен был это отфильтровать.
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
