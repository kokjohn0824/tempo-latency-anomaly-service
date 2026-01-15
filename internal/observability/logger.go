package observability

import "log"

// SetupLogger configures the standard logger with useful flags.
func SetupLogger() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
}

