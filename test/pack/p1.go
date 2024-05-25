package p

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Retrieve the verbosity level from an environment variable
func getVerbosity() int {
	v := os.Getenv("VERBOSE")
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid verbosity %v", v)
		}
	}
	return level
}

var debugStart time.Time

func init() {
	debugStart = time.Now()
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	//log.SetFlags(log.Ltime | log.Lmicroseconds)
	if Debug {
		t := time.Since(debugStart).Microseconds()
		t /= 100
		prefix := fmt.Sprintf("%06d  ", t)
		format = prefix + format
		log.Printf(format, a...)
	}
}
