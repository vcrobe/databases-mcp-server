package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

// wiretapStdio physically intercepts standard input and output to log all JSON-RPC traffic.
func wiretapStdio(logger *log.Logger) {
	// --- 1. Wiretap Stdin (Incoming from Copilot) ---
	originalStdin := os.Stdin
	rIn, wIn, _ := os.Pipe()
	os.Stdin = rIn // Trick the SDK into reading from our pipe

	go func() {
		reader := bufio.NewReader(originalStdin)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				// Pretty-print the raw JSON-RPC payload
				var rawJSON map[string]any
				if json.Unmarshal(line, &rawJSON) == nil {
					pretty, _ := json.MarshalIndent(rawJSON, "", "  ")
					logger.Printf("<= INCOMING (Copilot):\n%s\n", string(pretty))
				} else {
					logger.Printf("<= INCOMING (Copilot):\n%s\n", string(line))
				}
				wIn.Write(line) // Forward to the SDK
			}
			if err != nil {
				break
			}
		}
	}()

	// --- 2. Wiretap Stdout (Outgoing from Server) ---
	originalStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut // Trick the SDK into writing to our pipe

	go func() {
		reader := bufio.NewReader(rOut)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				var rawJSON map[string]any
				if json.Unmarshal(line, &rawJSON) == nil {
					pretty, _ := json.MarshalIndent(rawJSON, "", "  ")
					logger.Printf("=> OUTGOING (Server):\n%s\n", string(pretty))
				} else {
					logger.Printf("=> OUTGOING (Server):\n%s\n", string(line))
				}
				originalStdout.Write(line) // Forward to Copilot
			}
			if err != nil {
				break
			}
		}
	}()
}
