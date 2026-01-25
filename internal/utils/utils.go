package utils

import (
	"log"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("error env loading %v", err.Error())
	}
}
