package parser

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
    colorReset  = "\033[0m"
    colorRed    = "\033[31m"
    colorYellow = "\033[33m"
    colorGreen  = "\033[32m"
)

func colorWarning(message string) string {
    return fmt.Sprintf("%sWARNING: %s%s", colorYellow, message, colorReset)
}

type ConfigParser interface {
	Parse(filepath string) (map[string]string, error)
	Write(filepath string, config map[string]string) error
	ConvertToGhostty(config map[string]string) (map[string]string, error)
}

type KittyParser struct {
	configPath string
}
type AlacrittyParser struct {
	isParsingTheme bool
	recursionDepth int
	maxRecursionDepth int
	configPath string
}

// sort keys alphabetically
func sortKeysAlphabetically(config map[string]string) []string {
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// kittywriter
func (p *KittyParser) Write(filepath string, config map[string]string) error {
	 // Create directory if it doesn't exist
	 dir := path.Dir(filepath)
	 if err := os.MkdirAll(dir, 0755); err != nil {
		 return fmt.Errorf("failed to create directory: %w", err)
	 }

	// before writing the file backeup the old one if there is one
	if _, err := os.Stat(filepath); err == nil {
		backupPath := filepath + ".bak"
		  // Remove existing backup if it exists
		  if _, err := os.Stat(backupPath); err == nil {
			if err := os.Remove(backupPath); err != nil {
				return fmt.Errorf("failed to remove existing backup: %w", err)
			}
		}
		
		// Create new backup
		if err := os.Rename(filepath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// sort the keys alphabetically
	keys := sortKeysAlphabetically(config)

	// write to the file
	writer := bufio.NewWriter(file)
	for _, key := range keys {
		// and = is already added to pallete in the func ConvertToGhostty
		if strings.Contains(key, "palette") {
			_, err := writer.WriteString(fmt.Sprintf("%s  %s\n", key, config[key]))
			if err != nil {
				return err
			}
		}else {
			_, err := writer.WriteString(fmt.Sprintf("%s = %s\n", key, config[key]))
			if err != nil {
				return err
			}
		}
	}
	return writer.Flush()
}


// method to get the appriopriate parser
func GetParser(terminalType string, configPath string) (ConfigParser, error) {
	// make terminal type all lowercase
	terminalType = strings.ToLower(terminalType)
	switch terminalType {
	case "kitty":
		return NewKittyParser(configPath), nil
	case "alacritty":
		return NewAlacrittyParser(configPath), nil
	default:
		return nil, fmt.Errorf("unsupported terminal type: %s", terminalType)
	}
}

// a constructer for kittyParser
func NewKittyParser(configPath string) *KittyParser {
	return &KittyParser{configPath: configPath}
}

// constructor for alacrittyParser
func NewAlacrittyParser(configPath string) *AlacrittyParser {
	return &AlacrittyParser{
		configPath: configPath,
		maxRecursionDepth: 2,
		recursionDepth: 0,
		isParsingTheme: false,
	}
}

//  Parse the config file
func (p *KittyParser) Parse(filepath string) (map[string]string, error) {
    config := make(map[string]string)
    file, err := os.Open(filepath)
    if err != nil {
        return nil, fmt.Errorf("failed to open config file: %w", err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0
    
    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        // Skip empty lines and comments
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        // Split on first space only to handle multi-word values
        parts := strings.SplitN(line, " ", 2)
        if len(parts) != 2 {
            continue // or return error if strict parsing needed
        }

        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        
        // Only remove matching quotes
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		value = value[1 : len(value)-1]
		}
        
        if key == "" {
            return nil, fmt.Errorf("empty key found at line %d", lineNum)
        }

        config[key] = value
    }

    // Check for scanner errors
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("error reading config: %w", err)
    }

    return config, nil
}

func (p *KittyParser) ConvertToGhostty(kittyConfig map[string]string) (map[string]string, error) {
	ghosttyConfig := make(map[string]string)
	var unmappedKeys []string

	for kittyKey, value := range kittyConfig {
		
        if ghosttyKey, exists := kittyToGhosttyCodex[kittyKey]; exists {
			
            ghosttyConfig[ghosttyKey] = value
        }else {
			// handle unmapped keys
			unmappedKeys = append(unmappedKeys, kittyKey)
		}
	}

	// handle theme conversion from kitty to ghostty
	for key, value := range kittyConfig {
		if key == "include" && strings.Contains(value, "themes/") {
			// split the path to get the theme name
			parts := strings.Split(value, "/")
			if len(parts) > 0 {
				// get the last part and remove the extension
				themeName := parts[len(parts)-1]
				themeName = strings.TrimSuffix(themeName, ".conf")
				ghosttyConfig["theme"] = themeName
			}
		}else if strings.Contains(value, "current-theme.conf") {
			fmt.Println("Debug - Theme is a current theme file")
			// parse the theme file
			themeConfig := p.parseKittyThemeFile(value)
			
			// add all the values to the ghostty config
			for key, value := range themeConfig {
				ghosttyConfig[key] = value
			}
		}
	}

	// add unmapped keys to the ghostty config but comment them out
	// add a cmment saying that they are unmapped settings
	ghosttyConfig["# Unmapped settings"] = ""
	for _, key := range unmappedKeys {
		ghosttyConfig["# " + key] = kittyConfig[key]
	}

	

	return ghosttyConfig, nil
}


// helper function to parse kitty theme file
func (p *KittyParser) parseKittyThemeFile(themeFile string) map[string]string {
	configDir := filepath.Dir(p.configPath)
	fullThemePath := filepath.Join(configDir, themeFile)
	fmt.Printf("Debug - Full theme path: %s\n", fullThemePath)

	themeConfig := make(map[string]string)

	// get the contents of the theme file
	file, err := os.Open(fullThemePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening theme file: %v\n", err)
		fmt.Print("Theme file not found\n")
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text()) 
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			fmt.Print("Empty key found in theme file, ignored\n")
			continue
		}

		themeConfig[key] = value
	}

	// check for scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading theme file: %v\n", err)
		return nil
	}


return ConvertKittyThemeToGhostty(themeConfig)

}

// handle the kitty theme conversion
func ConvertKittyThemeToGhostty(themeFile map[string]string) map[string]string {

	ghosttyThemeConfig := make(map[string]string)
	var unmappedKeys []string

	for kittyKey, value := range themeFile {
		if ghosttyKey, exists := kittyToGhosttyThemeCodex[kittyKey]; exists {
			// format the ghostty key correctly eg palette = 26=#005fd7
			if strings.Contains(ghosttyKey, " = ") {
				parts := strings.SplitN(ghosttyKey, " = ", 2)
				baseKey := parts[0]
				modifier := parts[1]

				// combine with the value
				ghosttyThemeConfig[baseKey + " = " + modifier] = value
			}else {
				ghosttyThemeConfig[ghosttyKey] = value
			}
		}else {
			// hanndle unmapped keys
			unmappedKeys = append(unmappedKeys, kittyKey)
		}
	}

	// add unmapped keys to the ghostty theme config but comment them out
	ghosttyThemeConfig["# Unmapped settings"] = ""
	for _, key := range unmappedKeys {
		ghosttyThemeConfig["# " + key] = themeFile[key]
	}

	// return the ghostty theme config
	return ghosttyThemeConfig
}

	// alacritty
	// Implement the Parse method
func (a *AlacrittyParser) Parse(SourceFilepath string) (map[string]string, error) {
	configDir := filepath.Dir(a.configPath)
	// handle recursion
	if a.recursionDepth > a.maxRecursionDepth {
		return nil, fmt.Errorf("maximum recursion depth reached")
	}
	a.recursionDepth++
	defer func() { a.recursionDepth-- }()
	
	// Add your implementation here
	file, err := os.Open(SourceFilepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()
	
	config := make(map[string]string)
	var fullThemePath string
	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// handle the theme file
		if strings.Contains(line, "/theme/") || 
    	strings.Contains(line, "themes") && 
    	strings.HasSuffix(line, `.toml",`) {
    
			// Clean and normalize path
			themePath := strings.Trim(line, `"',`)
			
			// Get home directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf(colorWarning("Could not get home directory: %v\n"), err)
				continue
			}
			
			// Handle ~ expansion and relative paths
			if strings.HasPrefix(themePath, "~") {
				themePath = filepath.Join(homeDir, themePath[1:])
			} else if strings.HasPrefix(themePath, ".") ||
			strings.HasPrefix(themePath, "/") {
				themePath = filepath.Join(configDir, themePath)
			} else if !filepath.IsAbs(themePath) {
				themePath = filepath.Join(homeDir, themePath)
			}
			
			fullThemePath = filepath.Clean(themePath)
			continue
		}
		

		// because toml, handle section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}

		// parse the key value pairs
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0]) 
			value := strings.TrimSpace(parts[1])

			// remove matching quotes
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}

			// handle nested keys
			if currentSection != "" {
				key = a.normaliseAlacrittyKey(currentSection + "." + key) 
			}

			// normalise the value
			value = a.normaliseAlacrittyValue(value)

			config[key] = value
		}
	}

	// add the theme file values if there was one
	if fullThemePath != "" && !a.isParsingTheme {
		a.isParsingTheme = true
		// check that the file exists
		if _, err := os.Stat(fullThemePath); err != nil {
			fmt.Printf(colorWarning("Theme file not found: %v\n"), err)
		}
		themeConfig, err := a.Parse(fullThemePath)
		if err != nil {
			fmt.Printf(colorWarning("Failed to parse theme file: %v\n"), err)
		}else {
			for key, value := range themeConfig {
				// only add colors
				if strings.Contains(key, "colors") {
					config[key] = value
				}

			}
		}
		a.isParsingTheme = false
	}

    return config, nil
}

// normalise alaraity keys
func (a *AlacrittyParser) normaliseAlacrittyKey(key string) string {
	// replace dots with underscores
	key = strings.ReplaceAll(key, ".", "_")

	// remove complex keys
	if strings.Contains(key, "{") {
		key = strings.Split(key, "{")[0]
	}

	if strings.Contains(key, "[") {
		key = strings.Split(key, "[")[0]
	}
	return key
}

// normalise alacritty values
func (a *AlacrittyParser) normaliseAlacrittyValue(value string) string {

	// handle fonts
	if strings.Contains(value, "family =") {
		if parts := strings.Split(value, "family ="); len(parts) > 1 {
			family := strings.TrimSpace(parts[1])
			// get the font name
			if idx := strings.Index(family, "\""); idx >= 0 {
                if endIdx := strings.Index(family[idx+1:], "\""); endIdx >= 0 {
                    return family[idx+1 : idx+1+endIdx]
                }
            }
		}
	}

	// handle window padding
	if strings.Contains(value, "x") {
		if parts := strings.Split(value, "x ="); len(parts) > 1 {
			if len(parts) > 1 {
				xValue := strings.Split(strings.TrimSpace(parts[1]), ",")[0]
				return xValue
			}
		}
	}

	// make Always or always or on to true
	if strings.ToLower(value) == "always" || strings.ToLower(value) == "on" || strings.ToLower(value) == "full" {
		return "true"
	}

	// neveer to false
	if strings.ToLower(value) == "never" || strings.ToLower(value) == "off" {
		return "false"
	}

	// OnlyCopy
	if strings.ToLower(value) == "onlycopy" {
		return "true"
	}

	return value
}

// Implement the Write method
func (a *AlacrittyParser) Write(filepath string, config map[string]string) error {
	 // Create directory if it doesn't exist
	 dir := path.Dir(filepath)
	 if err := os.MkdirAll(dir, 0755); err != nil {
		 return fmt.Errorf("failed to create directory: %w", err)
	 }

    //  before writing the file make a backup if ther is already one
	if _, err := os.Stat(filepath); err == nil {
		backupPath := filepath + ".bak"
		// Remove existing backup if it exists
		if _, err := os.Stat(backupPath); err == nil {
			if err := os.Remove(backupPath); err != nil {
				return fmt.Errorf("failed to remove existing backup: %w", err)
			}
		}

		// Create new backup
		if err := os.Rename(filepath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// create the file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// sort the keys alphabetically
	keys := sortKeysAlphabetically(config)

	// write to the file
	writer := bufio.NewWriter(file)
	for _, key := range keys {
		_, err := writer.WriteString(fmt.Sprintf("%s = %s\n", key, config[key]))
		if err != nil {
			return err
		}
	}


    return writer.Flush()
}

func (a *AlacrittyParser) ConvertToGhostty(config map[string]string) (map[string]string, error) {
	ghosttyConfig := make(map[string]string)

	for alacrittyKey, value := range config {
		if ghosttyKey, exists := alacrittyToGhostty[alacrittyKey]; exists {
			ghosttyConfig[ghosttyKey] = value
		}else {
			// handle unmapped keys
			ghosttyConfig["# " + alacrittyKey] = value
		}
	}

	// if alacritty has a blur set it to default 10 in ghostty
	for key, value := range config {
		// if key window_blur is value true set it to 10
		if key == "window_blur" && value == "true" {
			ghosttyConfig["background-blur-radius"] = "10"
		}
	}

	// print the convert congif
	// for key, value := range ghosttyConfig {
	// 	fmt.Printf("DEBUG GC - %s %s\n", key, value)
	// }

	return ghosttyConfig, nil
}
	



var kittyToGhosttyThemeCodex = map[string]string{
	 // Standard colors
	 "background": "background",
	 "foreground": "foreground",
	 "cursor": "cursor-color",
	 "selection_background": "selection-background",
	 "selection_foreground": "selection-foreground",
	 
	 // Color palette
	 "color0": "palette = 0=",
	 "color1": "palette = 1=",
	 "color2": "palette = 2=",
	 "color3": "palette = 3=",
	 "color4": "palette = 4=",
	 "color5": "palette = 5=",
	 "color6": "palette = 6=",
	 "color7": "palette = 7=",
	 "color8": "palette = 8=",
	 "color9": "palette = 9=",
	 "color10": "palette = 10=",
	 "color11": "palette = 11=",
	 "color12": "palette = 12=",
	 "color13": "palette = 13=",
	 "color14": "palette = 14=",
	 "color15": "palette = 15=",
}

// create the mapping functions
var kittyToGhosttyCodex = map[string]string{
	  // Font settings
	  "font_family": "font-family",
	  "bold_font": "font-family-bold",
	  "italic_font": "font-family-italic",
	  "bold_italic_font": "font-family-bold-italic",
	  "font_size": "font-size",
	  
	  // Window settings
	  "window_padding": "window-padding",
	  "remember_window_size": "window-save-state",
	  "initial_window_width": "window-width",
	  "initial_window_height": "window-height",
	  "window_resize_step_cells": "window-resize-step",
	  "window_decorations": "window-decoration",
	  "window_opacity": "background-opacity",

		// scrolling
		"scrolling_multiplier": "mouse-scroll-multiplier",

		// mouse
		"mouse_hide_when_typing": "mouse-hide-while-typing",

	  
	  // Terminal behavior
	  "selection_save_to_clipboard": "copy-on-select",
	  
	  
	  // Cursor
	  "cursor_shape": "cursor-style",
	  "cursor_beam_thickness": "cursor-beam-width",
	  "cursor_blink_interval": "cursor-blink-interval",
	  "colors_cursor_cursor": "cursor-color",
		"colors_cursor_text": "cursor-text-color",
	  
	  // MacOS specific
	  "macos_option_as_alt": "macos-option-as-alt",
	  "macos_titlebar_color": "macos-titlebar-style",
	  "macos_window_resizable": "window-resize-from-any-edge",
	  
	  // Shell integration
	  "shell": "command",
	  "working_directory": "working-directory",

	   // Basic colors
	   "colors_primary_background": "background",
	   "colors_primary_foreground": "foreground",
	   
	   
	   // Selection colors
	   "colors_selection_background": "selection-background",
	   "colors_selection_text": "selection-foreground",
	   
	   
  
	  // Additional mappings from config
	  "tab_bar_edge": "gtk-tabs-location",
	  "tab_bar_style": "adw-toolbar-style",
	  "scrollback_lines": "scrollback-limit",
	  "repaint_delay": "window-vsync",
	  "input_delay": "window-vsync",
	  "window_alert_on_bell": "desktop-notifications",
	  "window_logo_position": "resize-overlay-position",
	  "window_padding_balance": "window-padding-balance",
	  "placement_strategy": "window-theme",
	
}



var alacrittyToGhostty = map[string]string{
	  // Font Settings
	  "font_normal": "font-family",
	  "font_size": "font-size",
	  "font_italic": "font-family-italic",
	  "font_bold_italic": "font-family-bold-italic",
	  "font_bold": "font-family-bold",

	  // Cursor Settings
	  "cursor-style": "cursor-style",
	  "cursor_vi_mode_style_blinking": "cursor-style-blink",
	  "cursor_text": "cursor-text",
	  "cursor_color": "cursor-color",
	  "cursor_blink": "cursor-style-blink",
	  "colors_cursor_cursor": "cursor-color",
	  "cursor_vi_mode_style_shape": "cursor-style",
	  "cursor_style_blinking": "cursor-style-blink",
	  "colors_cursor_text": "cursor-text",

	  // Colors
	  "colors_primary_background": "background",
	  "colors_primary_foreground": "foreground",
	  "colors_selection_foreground": "selection-foreground",
	  "colors_selection_background": "selection-background",
  
	  // Window Layout
	  "window_padding": "window-padding-x",
	  "window_padding_color": "window-padding-color",
	  "window_title": "title",
	  "window_decorations": "window-decoration",
	  
  
	  // Window Behavior
	  "window_inherit_working_directory": "window-inherit-working-directory",
	  "window_inherit_font_size": "window-inherit-font-size",
	  "window_save_state": "window-save-state",
	  "window_step_resize": "window-step-resize",
	  "window_new_tab_position": "window-new-tab-position",
	  "window_opacity": "background-opacity",
  
	  // Scrollback & Mouse
	  "scrolling_history": "scrollback-limit",
	  "keyboard_CopySelection": "copy-on-select",
  
	  // Clipboard Handling
	  "terminal_osc52": "clipboard-read",

	  // Normal colors (0-7)
	  "colors_normal_black": "palette = 0",
	  "colors_normal_red": "palette = 1",
	  "colors_normal_green": "palette = 2",
	  "colors_normal_yellow": "palette = 3",
	  "colors_normal_blue": "palette = 4",
	  "colors_normal_magenta": "palette = 5",
	  "colors_normal_cyan": "palette = 6",
	  "colors_normal_white": "palette = 7",
	  
	  // Bright colors (8-15)
	  "colors_bright_black": "palette = 8",
	  "colors_bright_red": "palette = 9",
	  "colors_bright_green": "palette = 10",
	  "colors_bright_yellow": "palette = 11",
	  "colors_bright_blue": "palette = 12",
	  "colors_bright_magenta": "palette = 13",
	  "colors_bright_cyan": "palette = 14",
	   "colors_bright_white": "palette = 15",
}

