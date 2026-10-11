// Dominoes: in a terminal, or in Concord channels.
//
// Run it from a terminal to play against the computer (up to three of it).
// Launched by a Concord server (which sets
// CONCORD_* variables), the same program hosts dominoes tables in the
// server's channels.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/table"
	"github.com/JMThomas00/concord-dominoes/game"
)

func main() {
	cfg, underConcord := plugin.ConfigFromEnv()
	if !underConcord {
		if err := table.RunLocal(game.Rules, nil); err != nil {
			log.Fatal(err)
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := plugin.Run(ctx, cfg, table.New(game.Rules).Handler()); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
