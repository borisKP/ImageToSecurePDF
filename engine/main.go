package main

import (
	"fmt"
	"os"
)

const usage = `ImageToSecurePDF — shared processing engine

The desktop apps launch this binary as a child process. It never opens a
browser and never listens on a socket.

Subcommands (JSON lines on stdout):
  list    <flags>   enumerate images in a folder / ZIP / 7z / RAR archive
  thumb   <flags>   decode one image into a temporary preview thumbnail
  build   <flags>   build the compressed, optionally encrypted PDF
  version           print product name and version as JSON

Automation / CLI:
  -cli -src <path> [-archive-pass <pw>] [-pdf-pass <pw>] -out <file.pdf>
       [-target-mb 20] [-max-edge 1600] [-page fit|image|paper] [-compression-mode target-mb|preset|lossless]

Send {"type":"cancel"} on stdin to cancel a running build.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			os.Exit(runList(os.Args[2:]))
		case "thumb":
			os.Exit(runThumb(os.Args[2:]))
		case "build":
			os.Exit(runBuild(os.Args[2:]))
		case "version":
			emit(map[string]any{"type": "version", "product": ProductName, "version": AppVersion})
			return
		case "help", "-h", "--help":
			fmt.Print(usage)
			return
		}
	}
	fmt.Print(usage)
}
