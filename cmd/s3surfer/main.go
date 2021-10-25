package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/cli/safeexec"

	"github.com/hirose31/s3surfer/pkg/c"
)

const version = "0.9.0"

var revision = "HEAD"

type buildInfo struct {
	Version  string
	Revision string
}

func (b buildInfo) String() string {
	return fmt.Sprintf(
		"s3surfer %s (rev: %s/%s)",
		b.Version,
		b.Revision,
		runtime.Version(),
	)
}

type CLI struct {
	Debug   string           `help:"write debug log info file" short:"d" type:"path"`
	Version kong.VersionFlag `help:"print version information and exit"`

	Bucket string `help:"S3 bucket name" short:"b" optional`
}

func init() {
	// https://github.com/rivo/tview/wiki/FAQ#why-do-my-borders-look-weird
	if os.Getenv("LC_CTYPE") != "en_US.UTF-8" {
		os.Setenv("LC_CTYPE", "en_US.UTF-8")
		env := os.Environ()
		argv0, err := safeexec.LookPath(os.Args[0])
		if err != nil {
			panic(err)
		}
		os.Args[0] = argv0
		if err := syscall.Exec(argv0, os.Args, env); err != nil {
			panic(err)
		}
	}
}

func main() {
	buildInfo := buildInfo{
		Version:  version,
		Revision: revision,
	}

	cli := CLI{}

	ctx := kong.Parse(&cli,
		kong.Name("s3surfer"),
		kong.Description("s3surfer is CLI tool for browsing S3 bucket and download objects.\nhttps://github.com/hirose31/s3surfer"),
		kong.UsageOnError(),
		kong.Vars{
			"version": buildInfo.String(),
		},
	)

	err := c.NewController(
		cli.Bucket,
		cli.Debug,
		buildInfo.String(),
	).Run()

	ctx.FatalIfErrorf(err)
}
