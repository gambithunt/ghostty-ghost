package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ConfigParser interface {
	Parse(filepath string) (map[string]string, error)
	Write(filepath string, config map[string]string) error
	ConvertToGhostty(config map[string]string) (map[string]string, error)
}

type KittyParser struct {
	configPath string
}
type AlacrittyParser struct {}

// kittywriter
func (p *KittyParser) Write(filepath string, config map[string]string) error {
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

	writer := bufio.NewWriter(file)
	for key, value := range config {
		_, err := writer.WriteString(fmt.Sprintf("%s %s\n", key, value))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}


// method to get the appriopriate parser
func GetParser(terminalType string, configPath string) (ConfigParser, error) {
	switch terminalType {
	case "kitty":
		return NewKittyParser(configPath), nil
	case "alacritty":
		return &AlacrittyParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported terminal type: %s", terminalType)
	}
}

// a constructer for kittyParser
func NewKittyParser(configPath string) *KittyParser {
	return &KittyParser{configPath: configPath}
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

	fmt.Println("Parsed configuration:************")
	for key, value := range config {
		fmt.Printf("%s %s\n", key, value)
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
func (a *AlacrittyParser) Parse(filepath string) (map[string]string, error) {
    // Add your implementation here
    return make(map[string]string), nil
}

// Implement the Write method
func (a *AlacrittyParser) Write(filepath string, config map[string]string) error {
    // Add your implementation here
    return nil
}

func (a *AlacrittyParser) ConvertToGhostty(config map[string]string) (map[string]string, error) {
	// Add your implementation here
	return make(map[string]string), nil
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
	  "window_padding_width": "window-padding",
	  "remember_window_size": "window-save-state",
	  "initial_window_width": "window-width",
	  "initial_window_height": "window-height",
	  "window_resize_step_cells": "window-resize-step",
	  
	  // Terminal behavior
	  "sync_to_monitor": "window-vsync",
	  "enable_audio_bell": "bells",
	  "copy_on_select": "copy-on-select",
	  
	  // Colors and appearance
	  "background_opacity": "background-opacity",
	  "background": "background",
	  "foreground": "foreground",
	  "cursor_color": "cursor-color",
	  "selection_foreground": "selection-foreground",
	  "selection_background": "selection-background",
	  
	  // Cursor
	  "cursor_shape": "cursor-style",
	  "cursor_beam_thickness": "cursor-beam-width",
	  "cursor_blink_interval": "cursor-blink-interval",
	  
	  // MacOS specific
	  "macos_option_as_alt": "macos-option-as-alt",
	  "macos_titlebar_color": "macos-titlebar-style",
	  "macos_window_resizable": "window-resize-from-any-edge",
	  
	  // Shell integration
	  "shell": "command",
	  "working_directory": "working-directory",
  
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
	  "font": "font-family",
	  "font_size": "font-size",
  
	  // Colors
	  "background": "background",
	  "foreground": "foreground",
	  "selection_foreground": "selection-foreground",
	  "selection_background": "selection-background",
	  "selection_invert": "selection-invert-fg-bg",
  
	  // Cursor Settings
	  "cursor": "cursor-style",
	  "cursor_text": "cursor-text",
	  "cursor_color": "cursor-color",
	  "cursor_opacity": "cursor-opacity",
	  "cursor_blink": "cursor-style-blink",
  
	  // Window Layout
	  "window_padding_x": "window-padding-x",
	  "window_padding_y": "window-padding-y",
	  "window_padding_balance": "window-padding-balance",
	  "window_padding_color": "window-padding-color",
	  "window_vsync": "window-vsync",
	  "window_decoration": "window-decoration",
	  "window_theme": "window-theme",
	  "window_height": "window-height",
	  "window_width": "window-width",
  
	  // Window Behavior
	  "window_inherit_working_directory": "window-inherit-working-directory",
	  "window_inherit_font_size": "window-inherit-font-size",
	  "window_save_state": "window-save-state",
	  "window_step_resize": "window-step-resize",
	  "window_new_tab_position": "window-new-tab-position",
  
	  // Scrollback & Mouse
	  "scrollback": "scrollback-limit",
	  "mouse_bindings": "mouse-bindings",
	  "copy_on_select": "copy-on-select",
	  "click_repeat_interval": "click-repeat-interval",
	  "focus_follows_mouse": "focus-follows-mouse",
  
	  // Clipboard Handling
	  "clipboard_read": "clipboard-read",
	  "clipboard_write": "clipboard-write",
	  "clipboard_trim_trailing_spaces": "clipboard-trim-trailing-spaces",
	  "clipboard_paste_protection": "clipboard-paste-protection",
	  "clipboard_paste_bracketed_safe": "clipboard-paste-bracketed-safe",
  
	  // System Integration
	  "shell_integration": "shell-integration",
	  "shell_integration_features": "shell-integration-features",
	  "config_file": "config-file",
	  "config_default_files": "config-default-files",
  
	  // macOS Specific
	  "macos_non_native_fullscreen": "macos-non-native-fullscreen",
	  "macos_titlebar_style": "macos-titlebar-style",
	  "macos_titlebar_proxy_icon": "macos-titlebar-proxy-icon",
	  "macos_option_as_alt": "macos-option-as-alt",
	  "macos_window_shadow": "macos-window-shadow",
	  "macos_auto_secure_input": "macos-auto-secure-input",
	  "macos_secure_input_indication": "macos-secure-input-indication",
  
	  // Linux Specific
	  "linux_cgroup": "linux-cgroup",
	  "linux_cgroup_memory_limit": "linux-cgroup-memory-limit",
	  "linux_cgroup_processes_limit": "linux-cgroup-processes-limit",
	  "linux_cgroup_hard_fail": "linux-cgroup-hard-fail",
	  "gtk_single_instance": "gtk-single-instance",
	  "gtk_titlebar": "gtk-titlebar",
	  "gtk_tabs_location": "gtk-tabs-location",
	  "gtk_wide_tabs": "gtk-wide-tabs",
	  "gtk_adwaita": "gtk-adwaita",
}

