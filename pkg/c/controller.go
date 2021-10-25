package c

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/dustin/go-humanize"
	"github.com/gdamore/tcell/v2"
	"github.com/hirose31/s3surfer/pkg/m"
	"github.com/hirose31/s3surfer/pkg/v"
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/v3/disk"
)

type Controller struct {
	dfp *os.File
	v   v.View
	m   *m.S3Model
}

func NewController(
	bucket string,
	debug string,
	version string,
) Controller {

	var dfp *os.File
	if debug != "" {
		var err error
		if dfp, err = os.OpenFile(debug, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644); err != nil {
			panic(err)
		}
	}

	m := m.NewS3Model()
	if bucket != "" {
		m.SetBucket(bucket)
	}

	v := v.NewView()
	v.Frame.AddText(version, true, tview.AlignCenter, tcell.ColorWhite)

	c := Controller{
		dfp,
		v,
		m,
	}

	return c
}

func (c Controller) Debugf(format string, args ...interface{}) {
	fmt.Fprintf(c.dfp, format, args...)
}

func (c Controller) Run() error {
	c.Debugf(">> Run\n")
	c.Debugf("  bucket=%s\n", c.m.Bucket())

	if c.m.Bucket() != "" {
		c.m.SetBucket(c.m.Bucket())
	}

	c.updateList()

	c.setInputCapture()

	return c.v.App.Run()
}

func (c Controller) Stop() {
	c.v.App.Stop()
}

func (c Controller) setInputCapture() {
	c.v.App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				c.Stop()
				return nil
			}

		}
		return event
	})

	c.v.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'u', 'h':
				c.moveUp()
				return nil
			case 'j':
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case 'l':
				return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
			case 'd':
				i := c.v.List.GetCurrentItem()
				_, cur := c.v.List.GetItemText(i)
				cur = strings.TrimSpace(cur)
				c.Debugf("[%d] %s\n", i, cur)
				c.Debugf("download by d %s/%s%s\n", c.m.Bucket(), c.m.Prefix(), cur)
				c.Download(cur)
				return nil
			}

		}
		return event
	})
}

func (c Controller) updateList() {
	c.v.List.Clear()

	c.Debugf(">> updateList bucket=%s\n", c.m.Bucket())
	if c.m.Bucket() == "" {
		c.Debugf("select bucket\n")
		buckets := c.m.AvailableBuckets()
		c.Debugf("available buckets=%s\n", buckets)

		c.v.List.SetTitle("[ [::b]s3://[::-] ]")

		for _, _bucket := range buckets {
			bucket := _bucket.Name
			c.v.List.AddItem("[::b]s3://"+bucket+"[::-]", bucket, 0, func() {
				c.Debugf("select bucket=%s\n", bucket)

				c.m.SetBucket(bucket)
				c.updateList()
			})
		}
	} else {
		c.Debugf("select prefix or object\n")

		c.v.List.SetTitle("[ [::b]s3://" + c.m.Bucket() + "/" + c.m.Prefix() + "[::-] ]")

		prefixes, keys, err := c.m.List()
		if err != nil {
			panic(err)
		}
		c.Debugf("prefixes=%s keys=%s\n", prefixes, keys)

		for _, _prefix := range prefixes {
			prefix := _prefix
			c.v.List.AddItem("[::b]"+prefix+"[::-]", prefix, 0, func() {
				c.Debugf("select prefix=%s\n", prefix)
				c.moveDown(prefix)
			})
		}

		for _, _key := range keys {
			key := _key
			c.v.List.AddItem(key, key, 0, func() {
				c.Debugf("select key=%s\n", key)
				c.Debugf("download key %s/%s%s\n", c.m.Bucket(), c.m.Prefix(), key)
				c.Download(key)
			})
		}

	}
}

func (c Controller) moveUp() {
	c.Debugf("u1 %s/%s\n", c.m.Bucket(), c.m.Prefix())
	c.m.MoveUp()
	c.Debugf("u2 %s/%s\n", c.m.Bucket(), c.m.Prefix())
	c.updateList()
}

func (c Controller) moveDown(prefix string) {
	c.m.MoveDown(prefix)
	c.updateList()

}

func (c Controller) Download(key string) {
	c.Debugf("bucket=%s prefix=%s key=%s\n", c.m.Bucket(), c.m.Prefix(), key)

	totalSize := int64(0)
	existFilePath := []string{}
	objects := c.m.ListObjects(key)
	for _, object := range objects {
		filePath := aws.ToString(object.Key)
		c.Debugf("- %s\n", filePath)
		if _, err := os.Stat(filePath); err == nil {
			existFilePath = append(existFilePath, filePath)
		}
		totalSize += object.Size
	}

	// don't overwrite local files
	if len(existFilePath) > 0 {
		panic(fmt.Sprintf("\n[ABORT] following files are exists:\n%s\n", strings.Join(existFilePath, "\n")))
	}

	// check disk available
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	usage, err := disk.Usage(cwd)
	if err != nil {
		panic(err)
	}
	freeThreshold := int64(float64(usage.Free) * 0.8)
	c.Debugf("check disk free: totalSize=%d usage.Free=%d threshold=%d\n", totalSize, usage.Free, freeThreshold)
	if totalSize > freeThreshold {
		panic(fmt.Sprintf("[ABORT] there is not enough free space: download size=%d free=%d free threshold=%d", totalSize, usage.Free, freeThreshold))
	}

	nobjects := len(objects)

	progress := tview.NewModal().
		SetText("Downloading\n\n").
		AddButtons([]string{"Done"})

	confirm := tview.NewModal().
		SetText(fmt.Sprintf("Do you want to download?\n%d object(s)\ntotal size %s",
			nobjects,
			humanize.IBytes(uint64(totalSize)),
		)).
		AddButtons([]string{"OK", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			c.v.Pages.RemovePage("confirm").SwitchToPage("main")
			if buttonLabel == "OK" {
				c.v.Pages.AddAndSwitchToPage("progress", progress, true)

				go func() {
					downloadedSize := int64(0)
					title := "Downloading"

					for i, object := range objects {
						c.Debugf("download %s\n", aws.ToString(object.Key))
						n, err := c.m.Download(object)

						if err != nil {
							panic(err)
						}

						downloadedSize += n

						if i+1 == nobjects {
							title = "Downloaded"
						}

						c.v.App.QueueUpdateDraw(func() {
							progress.SetText(fmt.Sprintf("%s\n%d/%d objects\n%s/%s",
								title,
								i+1,
								nobjects,
								humanize.IBytes(uint64(downloadedSize)),
								humanize.IBytes(uint64(totalSize)),
							))
						})
					}

					progress.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						c.v.Pages.RemovePage("progress").SwitchToPage("main")
					})
				}()
			}
		})

	c.v.Pages.AddAndSwitchToPage("confirm", confirm, true)
}
