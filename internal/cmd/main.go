package cmd

import (
	"time"
)

// Http timeouts
const (
	ReadTimeout    = 30 * time.Second
	WriteTimeout   = 90 * time.Second
	IdleTimeout    = 120 * time.Second
	HandlerTimeout = 60 * time.Second
)
