package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Setup flags
	help := pflag.BoolP("help", "h", false, "Show this help message")
	pflag.StringSliceP("transfer", "t", []string{}, "Folders to transfer in src:dst format (e.g. -t /src:/dst)")
	pflag.BoolP("overwrite", "o", false, "Overwrite existing files at destination")
	pflag.StringP("hash", "a", "", "Hash algorithm to use (xxhash, blake2b, sha256, md5). Default: xxhash")
	pflag.IntP("buffer", "b", 0, "Buffer size in kilobytes (e.g. 1024 for 1MB). Default: 1MB")
	pflag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "CertiCopy - Secure File Transfer with Integrity Verification\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  certicopy [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  certicopy -t /path/to/src:/path/to/dst --hash blake2b --buffer 4096\n")
		os.Exit(0)
	}

	viper.BindPFlags(pflag.CommandLine)

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "certicopy",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
