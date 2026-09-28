// Chess: in a terminal, or in Concord channels.
//
// Run it from a terminal to play two-player (one keyboard), against the
// computer, or over the network. Launched by a Concord server (which sets
// CONCORD_* variables), the same program hosts chess tables in the
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
	"github.com/JMThomas00/concord-chess/game"
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
