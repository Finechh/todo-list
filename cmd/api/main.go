package main

import (
	"log"
	"todo_list/internal/app"
	"todo_list/internal/metrics"
)

func main() {
	metrics.Register()
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}

}
