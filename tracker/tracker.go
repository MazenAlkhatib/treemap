package tracker

import (
	"fmt"
	"time"
)

func TrackTime(start time.Time, functionName string) {
	fmt.Printf("%s took %s\n", functionName, time.Since(start))
}
