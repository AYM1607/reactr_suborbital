package main

import (
	"github.com/suborbital/reactr/api/tinygo/runnable"
	"github.com/suborbital/reactr/api/tinygo/runnable/cache"
)

type Cache struct{}

func (h Cache) Run(input []byte) ([]byte, error) {
	cache.Set("name", string(input), 0)

	return cache.Get("name")
}

func main() {
	runnable.Use(Cache{})
}
