package cmd

import (
	"log"
	"os"
)

// Check if the current user has root (superuser) privileges.
func isRoot() bool {
	// Get the effective UID of the current process
	uid := os.Getuid()
	return uid == 0
}

// Retrieve the home directory path of the current user.
func getHomeDirectory() string {
	// Retrieve the home directory path using the os.UserHomeDir function
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Error getting the user's home directory:", err)
	}
	return home
}
