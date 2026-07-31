package ui

import (
	"os"
	"path/filepath"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
	"github.com/kjkusap/monkeytype-clone/internal/progress"
	"github.com/kjkusap/monkeytype-clone/internal/shop"
)

// App wires persistence + progression + shop for a process.
type App struct {
	Store    *persist.Store
	Players  *player.Service
	Progress *progress.Service
	Shop     *shop.Service
}

// OpenApp opens the JSON store and services. path empty → GOTYPE_DATA_DIR or OS temp.
func OpenApp(path string) (*App, error) {
	if path == "" {
		path = os.Getenv("GOTYPE_DATA_DIR")
	}
	if path == "" {
		path = filepath.Join(os.TempDir(), "gotype", "data.json")
	} else if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		path = filepath.Join(path, "data.json")
	}
	store, err := persist.Open(path)
	if err != nil {
		return nil, err
	}
	players := player.NewService(store)
	prog := progress.NewService(store)
	inv := shop.NewInvoiceClient(shop.LNBitsFromEnv())
	return &App{
		Store:    store,
		Players:  players,
		Progress: prog,
		Shop:     shop.NewService(store, inv, prog),
	}, nil
}
