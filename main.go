package main

import (
	"bufio"
	"flag"
	"fmt"
	"ghostty-ghost/parser"
	"os"
	"path/filepath"
)

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
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
        os.Exit(1)
	}
	fmt.Print("Home directory: ", homeDir, "\n")

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
        {"Kitty", filepath.Join(homeDir, ".config", "kitty", "kitty.conf")},
        {"Alacritty", filepath.Join(homeDir, ".config", "alacritty", "alacritty.yml")},
    }

	// define the default paths
	defaultGhosttyPath := filepath.Join(homeDir, ".config", "ghostty","config")
	
	// Parse the flags
	fromTerminal := flag.String("f", "", "Terminal to convert from (k kitty, a alacritty)")
	sourcePath := flag.String("s", "", "Path to source terminal config")
	targetPath := flag.String("t", "defaultGhosttyPath", "Path to ghostty config")

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
		fmt.Fprintf(os.Stderr, "Invalid selection: %d\n", selection)
		os.Exit(1)
	}

	targetConfig := availableTerminals[selection-1]

	
	fmt.Printf("Target: %s\n", targetConfig.path)
	} else { // If flags have been used 

		// check if the source path exists
		if *sourcePath == "" {
			fmt.Fprintf(os.Stderr, "Source path not provided\n")
			os.Exit(1)

		}

		// check if the source path exists
		if !checkIfPathExists(*sourcePath) {
			fmt.Fprintf(os.Stderr, "Source path does not exist: %s\n", *sourcePath)
			os.Exit(1)
		}

		// check if the target path exists
		if *targetPath == "" {
			fmt.Fprintf(os.Stderr, "Target path not provided\n")
			os.Exit(1)
		}

		fmt.Printf("From: %s\n", *fromTerminal)
		fmt.Printf("Source: %s\n", *sourcePath)
		fmt.Printf("Defalt: %s\n", defaultGhosttyPath)
		fmt.Printf("Target: %s\n", *targetPath)

		// parse the config file
		// Get appropriate parser
		configParser, err := parser.GetParser(*fromTerminal, *sourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Parse config
		config, err := configParser.Parse(*sourcePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
			os.Exit(1)
		}

		// convert the config file to ghossty
		ghosttyConfig, err := configParser.ConvertToGhostty(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting config: %v\n", err)
			os.Exit(1)
		}

		// Print the converted config key by key
		// fmt.Println("Converted configuration: ----------------")
		// for key, value := range ghosttyConfig {
		// 	fmt.Printf("%s: %v\n", key, value)
		// }

		// write the ghossty config to the target path, if no target path is provided use the default path
		// Write the config
		if *targetPath == "" {
			*targetPath = defaultGhosttyPath
		}

		err = configParser.Write(*targetPath, ghosttyConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
			os.Exit(1)
		}

		// Print the target pathfmt.Printf("Successfully converted configuration to: %s\n", *targetPath)
		fmt.Println("You can now use this configuration with Ghostty terminal")
		os.Exit(0)		


		// Print the parsed config 
		fmt.Printf("Parsed config: %+v\n", config)

	}
	
}

// handle pasring the config file
func handleParseOfConfig (fromTerminal, sourcePath, targetPath, defaultGhosttyPath string) {
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
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// convert the config file to ghossty
	ghosttyConfig, err := configParser.ConvertToGhostty(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting config: %v\n", err)
		os.Exit(1)
	}

	// Print the converted config key by key
	fmt.Println("Converted configuration: ----------------")
	for key, value := range ghosttyConfig {
		fmt.Printf("%s: %v\n", key, value)
	}

	// write the ghossty config to the target path, if no target path is provided use the default path
	// Write the config
	if targetPath == "" {
		targetPath = defaultGhosttyPath
	}

	err = configParser.Write(targetPath, ghosttyConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}

	// Print the target pathfmt.Printf("Successfully converted configuration to: %s\n", targetPath)
	fmt.Println("You can now use this configuration with Ghostty terminal")
	os.Exit(0)

}