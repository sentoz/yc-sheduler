package validator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woozymasta/yc-scheduler/internal/yc"
	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

// Validator periodically inspects resources and logs their state.
// The current implementation only logs basic information and does
// not attempt to fix discrepancies.
type Validator struct {
	client *yc.Client
	cfg    *pkgconfig.Config
}

// New creates a new Validator instance.
func New(client *yc.Client, cfg *pkgconfig.Config) *Validator {
	return &Validator{
		client: client,
		cfg:    cfg,
	}
}

// Start runs validation in the background until the context is canceled.
func (v *Validator) Start(ctx context.Context, interval time.Duration) {
	if v == nil || v.client == nil || v.cfg == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				v.runOnce(ctx)
			}
		}
	}()
}

func (v *Validator) runOnce(ctx context.Context) {
	now := time.Now()

	for _, sch := range v.cfg.Schedules {
		log.Debug().
			Str("schedule", sch.Name).
			Str("resource_type", sch.Resource.Type).
			Str("resource_id", sch.Resource.ID).
			Time("now", now).
			Msg("Validator tick (state check not yet implemented)")
		// Здесь в будущем можно добавить реальные проверки состояния
		// ресурса через YC API и сравнение с ожидаемым состоянием.
	}
}
