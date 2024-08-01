package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println(os.Getenv("_DOGMA_BROWSER_GITHUB_TOKEN"))
}
