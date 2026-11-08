package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"time"

	"github.com/khambek-archakov/jitLog/internal/share"
)

func rowHasContent(img image.Image, y int, bgR, bgG, bgB uint32) bool {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, y).RGBA()
		dr := int(r>>8) - int(bgR)
		dg := int(g>>8) - int(bgG)
		db := int(bl>>8) - int(bgB)
		if dr*dr+dg*dg+db*db > 60*60 {
			return true
		}
	}
	return false
}

func main() {
	cases := []struct {
		name string
		in   share.Input
	}{
		{
			name: "dense",
			in: share.Input{
				Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
				GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
			},
		},
		{
			name: "single_no_rounds",
			in: share.Input{
				Name: "Мария", Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 1, TotalMinutes: 60, Rounds: 0,
				GiCount: 1,
			},
		},
	}

	for _, c := range cases {
		data, err := share.Render(c.in)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}

		path := "/tmp/share_safezone_" + c.name + ".png"
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		fmt.Println(path)

		if c.name == "dense" {
			img, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				panic(err)
			}
			bgR, bgG, bgB := uint32(244), uint32(241), uint32(234)

			b := img.Bounds()
			// Find rows with content, then find gaps (runs of rows with no content).
			var rows []bool
			for y := b.Min.Y; y < b.Max.Y; y++ {
				rows = append(rows, rowHasContent(img, y, bgR, bgG, bgB))
			}

			// find transitions from content->no-content->content in bottom half, print gap sizes
			inGap := false
			gapStart := 0
			for y := b.Max.Y / 2; y < len(rows); y++ {
				if !rows[y] && !inGap {
					inGap = true
					gapStart = y
				} else if rows[y] && inGap {
					inGap = false
					fmt.Printf("gap: y=%d to y=%d, height=%d\n", gapStart, y-1, y-gapStart)
				}
			}
		}
	}
}
