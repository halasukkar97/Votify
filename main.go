package main

import "votify/internal/app"

// main keeps Render's root build command working after the Go entrypoint moved to cmd/server.
func main() {
	app.Run()
}
