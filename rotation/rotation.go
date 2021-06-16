package rotation

import (
	"fmt"
)

func CollectFiles(iter string, nodeName string) bool {
	fmt.Printf("THIS IS ITER: %v and I'm %v\n", iter, nodeName)
	// Every time this is called, select a random quorum member
	// For loop for 3 files
		// ca.crt
		// nodeName.crt
		// nodeName.key
	return true
}
