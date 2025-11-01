package api

import (
	"chirpy/internal/database"
	"sync/atomic"
)

type Config struct {
	FileserverHits atomic.Int32
	DB             *database.Queries
	Platform       string
	JWTSecret	   string
	PolkaKey       string
}