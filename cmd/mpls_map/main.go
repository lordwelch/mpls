package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"timmy.narnian.us/mpls"
)

func main() {
	var (
		err     error
		dir     *os.File
		files   []string
		Seconds int64
	)
	flag.Int64Var(&Seconds, "s", 120, "Minimum duration of playlist")
	flag.Int64Var(&Seconds, "seconds", 120, "Minimum duration of playlist")
	flag.Parse()
	name := filepath.Join(flag.Arg(0), "BDMV", "PLAYLIST")
	dir, err = os.Open(name)
	if err != nil {
		panic(err)
	}
	files, err = dir.Readdirnames(0)
	if err != nil {
		panic(err)
	}
	for _, v := range files {
		var (
			file     *os.File
			playlist mpls.MPLS
			duration time.Duration
		)

		file, err = os.Open(filepath.Join(name, v))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		playlist, err = mpls.Parse(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if playlist.Duration > Seconds {
			duration = time.Duration(playlist.Duration) * time.Second
			fmt.Printf("%s %3d:%02d\n", v, int(duration.Minutes()), int(duration.Seconds())%60)

			fmt.Println(strings.Join(playlist.SegmentMap, ","))
		}
	}
}
