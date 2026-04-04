package main

import (
	"fmt"
	"os"
)

type DatabaseServerConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

var databaseServersConfig map[string]DatabaseServerConfig

func readDatabaseServersConfig() {
	databaseServersConfig = make(map[string]DatabaseServerConfig)

	for i := 0; ; i++ {
		serverNameKey := fmt.Sprintf("DATABASE_SERVER_NAME_%d", i)
		serverName := os.Getenv(serverNameKey)

		// Stop when no more servers are found
		if serverName == "" {
			break
		}

		// Read the corresponding configuration for this server
		hostKey := fmt.Sprintf("DATABASE_HOST_%d", i)
		portKey := fmt.Sprintf("DATABASE_PORT_%d", i)
		userKey := fmt.Sprintf("DATABASE_USER_%d", i)
		passwordKey := fmt.Sprintf("DATABASE_PASSWORD_%d", i)

		config := DatabaseServerConfig{
			Host:     os.Getenv(hostKey),
			Port:     os.Getenv(portKey),
			User:     os.Getenv(userKey),
			Password: os.Getenv(passwordKey),
		}

		// Add the server configuration to the map using the server name as key
		databaseServersConfig[serverName] = config
	}
}
