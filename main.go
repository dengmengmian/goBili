// goBili is a command-line tool for downloading videos from Bilibili.
// It supports single videos, multi-page videos, and playlists (bangumi),
// with quality selection, concurrent chunked downloads, and QR-code login.
//
// Usage:
//
//	goBili login           authenticate via QR code
//	goBili download <URL>  download a video or playlist
//	goBili version         print version information
package main

import (
	"fmt"
	"os"

	"github.com/dengmengmian/goBili/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
