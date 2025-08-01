package cmd

import (
	"os"

	"github.com/chrishrb/go-grip/pkg"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-grip [path]",
	Short: "Render markdown documents as html",
	Long:  `Render markdown documents as html. Can handle a single file or a directory of markdown files.`,
	Args:  cobra.MatchAll(cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		theme, _ := cmd.Flags().GetString("theme")
		browser, _ := cmd.Flags().GetBool("browser")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		boundingBox, _ := cmd.Flags().GetBool("bounding-box")

		var path string
		if len(args) == 1 {
			path = args[0]
		} else {
			path = "."
		}

		parser := pkg.NewParser(theme)
		server := pkg.NewServer(host, port, theme, boundingBox, browser, parser)
		return server.Serve(path)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String("theme", "auto", "Select css theme [light/dark/auto]")
	rootCmd.Flags().BoolP("browser", "b", true, "Open new browser tab")
	rootCmd.Flags().StringP("host", "H", "localhost", "Host to use")
	rootCmd.Flags().IntP("port", "p", 6419, "Port to use")
	rootCmd.Flags().Bool("bounding-box", true, "Add bounding box to HTML")
}
