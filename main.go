package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/pborman/getopt"

	// io/ioutil is deprecated, use io and os packages instead
	"notashelf.dev/hyprkeys/flags"
	"notashelf.dev/hyprkeys/reader"
)

func main() {
	flags := flags.ReadFlags()
	if flags.Version {
		version := "unknown"
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
		}
		fmt.Println("version:", version)
		return
	}
	if !(len(os.Args) > 1) || flags.Help {
		getopt.Usage()
		return
	}

	if flags.ConfigPath == "" {
		flags.ConfigPath = filepath.Join(os.Getenv("HOME"), ".config/hypr/hyprland.conf")
	}

	configValues, err := reader.ReadHyprlandConfig(flags)
	if err != nil {
		log.Println(err.Error())
		return
	}

	err = outputData(configValues, flags)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func outputData(configValues *reader.ConfigValues, flags *flags.Flags) error {
	configValues.Binds = filterBinds(configValues, flags)
	if flags.Markdown {
		return markdownHandler(configValues, flags)
	}
	if flags.Raw {
		return rawHandler(configValues, flags)
	}
	if flags.Json {
		return jsonHandler(configValues, flags)
	}
	return fmt.Errorf("No output flag selected")
}

func filterBinds(configValues *reader.ConfigValues, flags *flags.Flags) []*reader.Keybind {
	matchedBinds := make([]*reader.Keybind, 0)
	for _, val := range configValues.Binds {
		if strings.Contains(val.Dispatcher, flags.FilterBinds) || strings.Contains(val.Command, flags.FilterBinds) {
			matchedBinds = append(matchedBinds, val)
		}
	}
	return matchedBinds
}

func markdownHandler(configValues *reader.ConfigValues, flags *flags.Flags) error {
	md := keybindsToMarkdown(configValues.Binds)
	out := ""
	for _, val := range configValues.Keywords {
		out += fmt.Sprintf("#### $%s = %s", val.Name, val.Value)
	}
	out += "\n"
	out += "| Keybind | Dispatcher | Command | Comments |\n"
	out += "|---------|------------|---------|----------|\n"
	for _, row := range md {
		out += row + "\n"
	}
	fmt.Println(out)
	if flags.Output != "" {
		err := os.WriteFile(flags.Output, []byte(out), 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}

func jsonHandler(configValues *reader.ConfigValues, flags *flags.Flags) error {
	var out []byte
	var err error

	out, err = json.MarshalIndent(configValues, "", " ")
	if err != nil {
		return err
	}

	fmt.Println(string(out))
	if flags.Output != "" {
		err := os.WriteFile(flags.Output, out, 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}

func rawHandler(configValues *reader.ConfigValues, flags *flags.Flags) error {
	out := ""
	if flags.Variables {
		for _, val := range configValues.Settings {
			out += val.Name + " {" + "\n"
			for setting, value := range val.Settings {
				out += "\t" + setting + " = " + value + "\n"
			}
			for _, set := range val.SubCategories {
				out += "\t" + set.Name + " {\n"
				for setting, value := range set.Settings {
					out += "\t\t" + setting + " = " + value + "\n"
				}
				out += "\t}\n"
			}
			out += "}\n"
		}
	}
	if flags.AutoStart {
		for _, val := range configValues.AutoStart {
			out += fmt.Sprintf("%s=%s\n", val.ExecType, val.Command)
		}
	}
	for _, bind := range configValues.Binds {
		out += fmt.Sprintf("%s = %s %s %s", bind.BindType, bind.Bind, bind.Dispatcher, bind.Command)
		if bind.Comments != "" {
			out += fmt.Sprintf("#%s", bind.Comments)
		}
		out += "\n"
	}
	for _, key := range configValues.Keywords {
		out += fmt.Sprintf("$%s = %s\n", key.Name, key.Value)
	}
	fmt.Print(out)
	if flags.Output != "" {
		err := os.WriteFile(flags.Output, []byte(out), 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}

// Pass both kbKeybinds and mKeybinds to this function
func keybindsToMarkdown(binds []*reader.Keybind) []string {
	var markdown []string
	for _, keybind := range binds {
		markdown = append(markdown, "| <kbd>"+keybind.Bind+"</kbd> | "+keybind.Dispatcher+" | "+strings.ReplaceAll(keybind.Command, "|", "\\|")+" | "+strings.ReplaceAll(keybind.Comments, "|", "\\|")+" |")
	}
	return markdown
}
