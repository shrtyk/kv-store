package main

import (
	"github.com/shrtyk/kv-store/internal/server"
)

func main() {
	server.Serve(":16700", server.NewHandler())
}
