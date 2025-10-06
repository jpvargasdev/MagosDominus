package events

import (
	"log"
	"time"
)

type Event struct {
  File       string // Path to YAML File
  Repo       string // owner/name
  Ref        string // tag or Ref
  Digest     string // sha256...
  Policy     string // "semver", "latest", etc
  Discovered time.Time // When the event was discovered
}

type Emitter interface {
  Emit(e Event)
}

// Buffered channel emitter
type ChanEmitter chan Event

func (c ChanEmitter) Emit(e Event) {
  select {
    case c <- e:
      // sent
    default:
      // dropped
      log.Printf("[events] buffer null, dropping event %+v", e)
  }
}
