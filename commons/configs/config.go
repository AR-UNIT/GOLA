package configs

// Config
type Config struct {
	Port          string
	JWTSecret     string
	DBConnection  string
	ImageStorage  string
	MaxImageSize  int64
	AllowedTypes  []string
	TokenValidity int
}
