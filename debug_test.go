package main

import (
	"fmt"

	"github.com/gomdlint/gomdlint/pkg/gomdlint/helpers"
)

func main() {
	content := `---
title: Test
author: Someone
---

# Main Content`

	fm, body, hasFM := helpers.ExtractFrontMatter(content)
	fmt.Printf("hasFM: %v\n", hasFM)
	fmt.Printf("fm: %q\n", fm)
	fmt.Printf("body: %q\n", body)
	fmt.Printf("body starts with '# Main Content': %v\n", body == "# Main Content")
}
