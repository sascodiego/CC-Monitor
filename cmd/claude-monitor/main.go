/**
 * CONTEXT:   Claude Monitor unified binary entry point
 * INPUT:     Command line arguments and system environment
 * OUTPUT:    Application execution with proper error handling and exit codes
 * BUSINESS:  Single entry point simplifies deployment and user interaction
 * CHANGE:    Simplified main.go after extracting concerns to separate files
 * RISK:      Low - Entry point with delegated responsibilities
 */

package main

import (
	"os"
)

/**
 * CONTEXT:   Application main entry point with error handling
 * INPUT:     Command line arguments via os.Args
 * OUTPUT:    Command execution with appropriate exit codes
 * BUSINESS:  Main function coordinates application startup and command routing
 * CHANGE:    Simplified main function delegating to extracted command system
 * RISK:      Low - Entry point with proper error handling and exit codes
 */
func main() {
	// Execute the root command and handle any errors
	if err := executeRootCommand(); err != nil {
		errorColor.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}