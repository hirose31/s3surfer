package c

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/hirose31/s3surfer/pkg/m"
	"github.com/hirose31/s3surfer/pkg/v"
)

type Controller struct {
	bucket string
	dfp    *os.File
	v      v.View
	m      m.S3Model
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
		m.NewS3Model(),
	}

	return c
}

func (c Controller) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(c.dfp, format, args...)
}

func (c Controller) Run() error {
	c.Debugf(">> Run\n")
	c.Debugf("  bucket=%s\n", c.bucket)

	if c.bucket != "" {
		c.m.SetBucket(c.bucket)
	}

	c.updateList()

	c.setInputCapture()

	return c.v.App.Run()
}

func (c Controller) setInputCapture() {
	c.v.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'u':
				c.Debugf("u1 %s/%s\n", c.m.Bucket(), c.m.Prefix())
				c.m.MoveUp()
				c.Debugf("u2 %s/%s\n", c.m.Bucket(), c.m.Prefix())
				c.updateList()
				return nil
			}
		}
		return event
	})
}

func (c Controller) updateList() {
	c.v.List.Clear()

	if c.bucket == "" {
		c.Debugf("select bucket\n")
		buckets := c.m.AvailableBuckets()
		c.Debugf("available buckets=%s\n", buckets)

		c.v.List.SetTitle("s3://")

		for _, _bucket := range buckets {
			bucket := _bucket
			c.v.List.AddItem(" "+bucket, "", 0, func() {
				c.Debugf("select bucket=%s\n", bucket)

				c.bucket = bucket
				c.m.SetBucket(bucket)
				c.updateList()
			})
		}
	} else {
		c.Debugf("select prefix or object\n")

		c.v.List.SetTitle("s3://" + c.m.Bucket() + "/" + c.m.Prefix())

		prefixes, objects, err := c.m.List()
		if err != nil {
			panic(err)
		}
		c.Debugf("prefixes=%s objects=%s\n", prefixes, objects)

		for _, _prefix := range prefixes {
			prefix := _prefix
			c.v.List.AddItem(" "+prefix, "", 0, func() {
				c.Debugf("select prefix=%s\n", prefix)

				c.m.MoveDown(prefix)
				c.updateList()
			})
		}

		for _, _object := range objects {
			object := _object
			c.v.List.AddItem(" "+object, "", 0, func() {
				c.Debugf("select object=%s\n", object)

				c.Debugf("download?\n") // fixme
			})
		}

	}
}
