package go_bagit

import (
	"fmt"
	"time"
	"runtime/debug"
)


var currentTime = time.Now()

var libraryVersion = "v0.1.1-alpha"

func GetSoftwareAgent() string {
	const url = "github.com/nyudlts/go-bagit"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Sprintf("go-bagit <%s>", url)
	}

	return fmt.Sprintf("go-bagit %s <%s>", info.Main.Version, url)
}
