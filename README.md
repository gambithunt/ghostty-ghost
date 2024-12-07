# Ghostty Ghost 🚀

A command-line tool to convert Kitty and Alacritty terminal configurations to Ghostty format.

## Features

- Convert Kitty terminal configurations to Ghostty format
- Convert Alacritty terminal configurations to Ghostty format
- Automatic backup of existing configuration files
- Interactive mode for easy configuration selection
- Support for theme conversion
- Alphabetically sorted configuration output
- Automatic color palette mapping

## Installation

```sh
brew tap ghostty/ghostty
brew install ghostty
```

Or download the latest release from the [releases page](#).

## Usage

### Interactive Mode

Simply run the tool without any arguments:

```sh
ghostty-ghost
```

This will:

- Detect available terminal configurations
- Let you select which configuration to convert
- Convert and save the configuration to Ghostty format

### Command Line Mode

```sh
ghostty-ghost [options]
```

#### Options:

- `-f, --from` Terminal to convert from ((k) kitty, (a) alacritty)
- `-s, --source` Path to source terminal config file
- `-t, --target` Path to target ghostty config file

#### Example:

```sh
ghostty-ghost -f kitty -s ~/.config/kitty/kitty.conf -t ~/.config/ghostty/config
```

## Additional Features

- Automatically creates backup files (.bak extension)
- Converts color schemes and themes
- Maintains comments for unmapped settings
- Creates target directory if it doesn't exist
- Provides colorized output for warnings and errors

## Configuration Path Defaults

- **Kitty:** `~/.config/kitty/kitty.conf`
- **Alacritty:** `~/.config/alacritty/alacritty.toml`
- **Ghostty:** `~/.config/ghostty/config`

## Contributing

Contributions are welcome! Feel free to submit issues and pull requests.

## License

This project is open source and available under the MIT License.
