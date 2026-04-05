package main

import (
	"fmt"
	"os"
	"strings"
)

type DatabaseServerConfig struct {
	Engine   string
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

var databaseServersConfig map[string]DatabaseServerConfig

func readDatabaseServersConfig() error {
	databaseServersConfig = make(map[string]DatabaseServerConfig)

	for i := 0; ; i++ {
		serverNameKey := fmt.Sprintf("DATABASE_SERVER_NAME_%d", i)
		serverName := strings.TrimSpace(os.Getenv(serverNameKey))

		// Stop when no more servers are found
		if serverName == "" {
			break
		}

		// Read the corresponding configuration for this server
		engineKey := fmt.Sprintf("DATABASE_SERVER_ENGINE_%d", i)
		hostKey := fmt.Sprintf("DATABASE_HOST_%d", i)
		portKey := fmt.Sprintf("DATABASE_PORT_%d", i)
		userKey := fmt.Sprintf("DATABASE_USER_%d", i)
		passwordKey := fmt.Sprintf("DATABASE_PASSWORD_%d", i)
		databaseKey := fmt.Sprintf("DATABASE_NAME_%d", i)

		engine := strings.ToLower(strings.TrimSpace(os.Getenv(engineKey)))
		if engine == "" {
			return fmt.Errorf("%s is required for server %q", engineKey, serverName)
		}
		if engine != "mysql" && engine != "postgres" {
			return fmt.Errorf("%s has invalid value %q for server %q (allowed: mysql, postgres)", engineKey, engine, serverName)
		}

		host := strings.TrimSpace(os.Getenv(hostKey))
		if host == "" {
			return fmt.Errorf("%s is required for server %q", hostKey, serverName)
		}

		user := strings.TrimSpace(os.Getenv(userKey))
		if user == "" {
			return fmt.Errorf("%s is required for server %q", userKey, serverName)
		}

		port := strings.TrimSpace(os.Getenv(portKey))
		if port == "" {
			return fmt.Errorf("%s is required for server %q", portKey, serverName)
		}

		password := os.Getenv(passwordKey)
		if strings.TrimSpace(password) == "" {
			return fmt.Errorf("%s is required for server %q", passwordKey, serverName)
		}

		database := strings.TrimSpace(os.Getenv(databaseKey))
		if engine == "postgres" && database == "" {
			return fmt.Errorf("%s is required for server %q when engine is postgres", databaseKey, serverName)
		}

		config := DatabaseServerConfig{
			Engine:   engine,
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			Database: database,
		}

		// Add the server configuration to the map using the server name as key
		databaseServersConfig[serverName] = config
	}

	if len(databaseServersConfig) == 0 {
		return fmt.Errorf("no database servers configured")
	}

	return nil
}
