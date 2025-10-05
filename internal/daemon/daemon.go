package daemon

import (
	"context"
	"log"
	"magos-dominus/internal/watcher"
)

type Daemon struct {
	cfg *watcher.Config
}

// New crea una nueva instancia del daemon con la configuración cargada.
func New(cfg *watcher.Config) *Daemon {
	return &Daemon{cfg: cfg}
}

// Start inicia el watcher principal.
func (d *Daemon) Start(ctx context.Context) error {
	log.Printf("[daemon] starting...")

	// Crear watcher con los targets definidos en la configuración
	w := watcher.New(d.cfg.Watcher.Targets)

	// Iniciar el ciclo de vigilancia
	return w.Start(ctx)
}
