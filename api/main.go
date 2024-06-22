package main

import (
	"github.com/Guilherme-Vale-98/encurtadorURL/routes"

	"github.com/gofiber/fiber/v3"
)

func setupRoutes(app *fiber.App){
	app.Get("/:url", routes.ResolveURL)
	app.Get("/api/v1", routes.ShortenURL)
}

func main() {

}