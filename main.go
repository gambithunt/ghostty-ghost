package main

import (
	"bufio"
	"flag"
	"fmt"
	"ghostty-ghost/parser"
	"os"
	"path/filepath"
)

const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorYellow = "\033[33m"
    colorGreen  = "\033[32m"
)

func colorError(message string) string {
	return fmt.Sprintf("%sERROR: %s%s", colorRed, message, colorReset)
}

func colorWarning(message string) string {
    return fmt.Sprintf("%sWARNING: %s%s", colorYellow, message, colorReset)
}

func colorSuccess(message string) string {
	return fmt.Sprintf("%s%s%s", colorGreen, message, colorReset)
}

type TerminalConfig struct {
	name string
	path string
}

func checkIfPathExists (path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func createConfigDir(path string) error {
	return os.Mkdir(filepath.Dir(path), 0755)
}	

func main() {
	// get the users home directory
	homeDir ,err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", colorError(fmt.Sprintf("Error getting home directory: %v", err)))
        os.Exit(1)
	}

	// Define custom usage text
    flag.Usage = func() {
        fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
        fmt.Println("Options:")
        fmt.Println("  -f, --from    Terminal to convert from ((k) kitty, (a) alacritty)")
        fmt.Println("  -s, --source  Path to source terminal config file")
        fmt.Println("  -t, --target  Path to target ghostty config file")
        fmt.Println("\nExample:")
        fmt.Printf("  %s -f kitty -s ~/.config/kitty/kitty.conf -t ~/.config/ghostty/config\n", os.Args[0])
        fmt.Println("\nIf no flags are specified, interactive mode will be used.")
    }

	 // Define known terminal configs
	 terminals := []TerminalConfig{
        {"kitty", filepath.Join(homeDir, ".config", "kitty", "kitty.conf")},
        {"alacritty", filepath.Join(homeDir, ".config", "alacritty", "alacritty.toml")},
    }

	// define the default paths
	defaultGhosttyPath := filepath.Join(homeDir, ".config", "ghostty","config")
	
	// Parse the flags
	fromTerminal := flag.String("f", "", "Terminal to convert from (k kitty, a alacritty)")
	sourcePath := flag.String("s", "", "Path to source terminal config")
	targetPath := flag.String("t", defaultGhosttyPath, "Path to ghostty config")

	flag.Parse()

	// check what is installed
	var availableTerminals []TerminalConfig
	for _, terminal := range terminals {
		if checkIfPathExists(terminal.path) {
			availableTerminals = append(availableTerminals, terminal)
		}
	}

	// if no flags have been used show the user the available terminals
	if *fromTerminal == "" {
		// let the user choose the terminal
		fmt.Println("Select available terminal config:")
		for i, terminal := range availableTerminals {
			fmt.Printf("%d. %s (%s)\n", i+1, terminal.name, terminal.path)
		}
	

		// get input from the user
		reader := bufio.NewReader(os.Stdin) 
		var selection int
		fmt.Print("Enter selection (1 -", len(availableTerminals), "): ")
		fmt.Fscanf(reader, "%d", &selection)

		if selection < 1 || selection > len(availableTerminals) {
			fmt.Println("\n❌ Invalid selection!")
			fmt.Println("\nAvailable options:")
			for i, term := range availableTerminals {
				fmt.Printf("  %d. %s\n", i+1, term)
			}
			fmt.Println("\nPlease try again with a valid number.")
			os.Exit(1)
		}

		targetConfig := availableTerminals[selection-1]
		fmt.Printf("Selected terminal: %s\n", targetConfig.name)
		fmt.Printf("Path: %s\n", targetConfig.path)

		handleParseOfConfig(targetConfig.name, targetConfig.path, *targetPath, defaultGhosttyPath)

		} else { // If flags have been used 

		handleParseOfConfig(*fromTerminal, *sourcePath, *targetPath, defaultGhosttyPath)


		}
		// Print the target pathfmt.Printf("Successfully converted configuration to: %s\n", *targetPath)
		fmt.Printf("\n%s %s\n", "✨", colorSuccess("Configuration converted successfully"))
		fmt.Printf("%s %s\n", "📝", "Configuration file saved to: " + *targetPath)
		fmt.Printf("%s %s\n\n", "🚀", "You can now use this configuration with Ghostty terminal")
		os.Exit(0)	
	
}

// handle pasring the config file
func handleParseOfConfig (fromTerminal, sourcePath, targetPath, defaultGhosttyPath string) {
	// do checks on the source path
	if sourcePath == "" {
		fmt.Fprintf(os.Stderr, "%s\n", colorWarning("The default source terminal path does not exist, please use ghostty-ghost -h for help"))
		os.Exit(1)
	}
	// check if the source path exists
	if !checkIfPathExists(sourcePath) {
		fmt.Fprintf(os.Stderr, "%s\n", colorWarning(fmt.Sprintf("The default terminal path does not exist: %s", sourcePath)))
		fmt.Println("Please use ghostty -h for help")
		os.Exit(1)
	}
	
	// if not target path is provided use the default path
	if targetPath == "" {
		targetPath = defaultGhosttyPath
	}
	
	// parse the config file
	// Get appropriate parser
	configParser, err := parser.GetParser(fromTerminal, sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse config
	config, err := configParser.Parse(sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", colorError(fmt.Sprintf("Error parsing config: %v\n", err)) )
		os.Exit(1)
	}

	// convert the config file to ghossty
	ghosttyConfig, err := configParser.ConvertToGhostty(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", colorError(fmt.Sprintf("Error converting config: %v\n", err)))
		os.Exit(1)
	}

	// write the ghossty config to the target path, if no target path is provided use the default path
	// Write the config
	if targetPath == "" {
		targetPath = defaultGhosttyPath
	}

	err = configParser.Write(targetPath, ghosttyConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", colorError(fmt.Sprintf("Error writing config: %v\n", err)))
		os.Exit(1)
	}

}