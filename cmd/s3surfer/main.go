package main

import (
	"fmt"
	"runtime"

	"github.com/alecthomas/kong"

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

func main() {
	buildInfo := buildInfo{
		Version:  version,
		Revision: revision,
	}

	cli := CLI{}

	ctx := kong.Parse(&cli,
		kong.Name("s3surfer"),
		kong.Description("s3surfer is CLI tool for browsing S3 bucket and download objects."),
		kong.UsageOnError(),
		kong.Vars{
			"version": buildInfo.String(),
		},
	)

	err := c.NewController(
		cli.Bucket,
		cli.Debug,
	).Run()

	ctx.FatalIfErrorf(err)
}
