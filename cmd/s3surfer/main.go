package main

import (
	"fmt"
	"os"
	"runtime"

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

var ()

func init() {
	//
}

func main() {
	buildInfo := buildInfo{
		Version:  version,
		Revision: revision,
	}

	fmt.Println(buildInfo.String())

	c.Blah()

	os.Exit(1)
}
