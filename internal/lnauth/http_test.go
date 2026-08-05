package lnauth

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"

	"github.com/kjkusap/monkeytype-clone/internal/persist"
	"github.com/kjkusap/monkeytype-clone/internal/player"
)

func TestHTTPCallbackAndStatus(t *testing.T) {
	t.Setenv("GOTYPE_PUBLIC_URL", "https://gotype.fun")
	t.Setenv("REDIS_URL", "")
	dir := t.TempDir()
	store, err := persist.Open(filepath.Join(dir, "gotype.json"))
	if err != nil {
		t.Fatal(err)
	}
	players := player.NewService(store)
	svc := NewService(players)
	mux := http.NewServeMux()
	NewHandler(svc).Mount(mux)

	start, err := svc.Start("sess-http", ActionLogin, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	msg, _ := hex.DecodeString(start.K1)
	sig := ecdsa.Sign(priv, msg)
	key := hex.EncodeToString(priv.PubKey().SerializeCompressed())

	req := httptest.NewRequest(http.MethodGet, "/auth/lnurl?tag=login&k1="+start.K1+
		"&sig="+hex.EncodeToString(sig.Serialize())+"&key="+key, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("callback status %d body %s", rec.Code, rec.Body.String())
	}

	stReq := httptest.NewRequest(http.MethodGet, "/auth/lnurl/status?k1="+start.K1, nil)
	stRec := httptest.NewRecorder()
	mux.ServeHTTP(stRec, stReq)
	if stRec.Code != http.StatusOK {
		t.Fatalf("status code %d", stRec.Code)
	}
	if !strings.Contains(stRec.Body.String(), `"verified"`) {
		t.Fatalf("body %s", stRec.Body.String())
	}
}
