package c

import (
	"fmt"
	"os"

	"github.com/hirose31/s3surfer/pkg/v"
)

type Controller struct {
	bucket string
	dfp    *os.File
	v      v.View
}

func NewController(
	bucket string,
	debug string,
) Controller {

	var dfp *os.File
	if debug != "" {
		var err error
		if dfp, err = os.OpenFile(debug, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			panic(err)
		}
	}

	c := Controller{
		bucket,
		dfp,
		v.NewView(),
	}

	return c
}

func (c Controller) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(c.dfp, format, args...)
}

func (c Controller) Run() error {
	c.Debugf(">> Run\n")
	c.Debugf("  bucket=%s\n", c.bucket)

	c.v.List.SetTitle("hhhhhhhhh")

	return c.v.App.Run()
}
