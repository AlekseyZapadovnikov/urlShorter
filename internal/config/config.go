package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Env        string     `json:"env"`
	HTTPServer HTTPServer `json:"http_server"`
	Storage    Storage    `json:"storage"`
}

type HTTPServer struct {
	Address     string `json:"address"`
	Timeout     int    `json:"timeout"`      // время ожидания ответа от сервера (секунды)
	IdleTimeout int    `json:"idle_timeout"` // время ожидания закрытия соединения (секунды)
}

type Storage struct {
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBName     string `json:"db_name"`
	DBPassword string `json:"-"`
	ServerPort string `json:"server_port"`
}

// MustLoad читает путь к файлу конфига из переменной окружения CONFIG_PATH,
// парсит JSON и возвращает указатель на Config.
// В случае ошибки — завершает работу с логом.
func MustLoad() *Config {
	var cfgPath string
	var ok bool
	cfgPath, ok = os.LookupEnv("CONFIG_PATH")
	if !ok {
		absWay := "C:/Users/Asus/Desktop/go_p/urlShorter/config/config.json"
		_, err := os.Stat(absWay) // я МЯГКО говоря не благодарен человеку, который посоветовал мне использовать $Env:CONFIG_PATH = ...
		if err != nil {
			log.Fatal("CONFIG_PATH environment variable is not set")
		} else {
			cfgPath = absWay
		}
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		log.Fatalf("config file %s does not exist", cfgPath)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("failed to parse config JSON: %v", err)
	}

	if pass, exists := os.LookupEnv("DB_PASSWORD"); exists {
        cfg.Storage.DBPassword = pass
    } else {
		if cfg.Env == "local" {
			cfg.Storage.DBPassword = "123"
		} else {
			log.Fatal("DB_PASSWORD environment variable is not set ")
		}
	}

	return &cfg
}


