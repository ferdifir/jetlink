package config

import "os"

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	SessionSecret string
	AppPort        string
	AppURL         string
	AppEnv         string
	FonnteToken    string
}

func Load() *Config {
	return &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "3306"),
		DBUser:        getEnv("DB_USER", "root"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "jetlink"),
		SessionSecret: getEnv("SESSION_SECRET", "change-me-in-production-32chars!!"),
		AppPort:       getEnv("APP_PORT", "8080"),
		AppURL:        getEnv("APP_URL", "http://localhost:8080"),
		AppEnv:        getEnv("APP_ENV", "development"),
		FonnteToken:    getEnv("FONNTE_TOKEN", ""),
	}
}

func (c *Config) IsDev() bool {
	return c.AppEnv == "development"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
