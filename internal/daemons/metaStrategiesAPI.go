package daemons

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/majorfi/ydaemon/internal/logs"
	"github.com/majorfi/ydaemon/internal/models"
	"github.com/majorfi/ydaemon/internal/store"
	"github.com/majorfi/ydaemon/internal/utils"
)

// FetchStrategiesFromMeta fetches the strategies information from the Yearn Meta API for a given chainID
// and store the result to the global variable StrategiesFromMeta for later use.
func FetchStrategiesFromMeta(chainID uint64) {
	// Get the meta information from the Yearn Meta API
	chainIDStr := strconv.FormatUint(chainID, 10)
	resp, err := http.Get(utils.META_BASE_URL + chainIDStr + `/strategies/all`)
	if err != nil {
		logs.Error("Error fetching meta information from the Yearn Meta API", err)
		return
	}

	// Defer the closing of the response body to avoid memory leaks
	defer resp.Body.Close()

	// Read the response body and store it in the body variable
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Error("Error reading response body from the Yearn Meta API", err)
		return
	}

	// Unmarshal the response body into the variable StrategiesFromMeta. Body is a byte array,
	// with this manipulation we are putting it in the correct TStrategyFromMeta struct format
	strategies := []models.TStrategyFromMeta{}
	if err := json.Unmarshal(body, &strategies); err != nil {
		logs.Error("Error unmarshalling response body from the Yearn Meta API", err)
		return
	}

	// To provide faster access to the data, we index the mapping by the vault address, aka
	// {[vaultAddress]: TStrategyFromMeta} if we were working with JS/TS
	if store.StrategiesFromMeta[chainID] == nil {
		store.StrategiesFromMeta[chainID] = make(map[string]models.TStrategyFromMeta)
	}
	for _, strategy := range strategies {
		for _, strategyAddress := range strategy.Addresses {
			// common.HexToAddress(strategyAddress).String() asserts that the address is a valid
			// chacksummed hex string
			store.StrategiesFromMeta[chainID][common.HexToAddress(strategyAddress).String()] = strategy
		}
	}
}

// RunMetaStrategies is a goroutine that periodically fetches the strategies information from the
// Yearn Meta API. It runs forever, every 15 seconds, for the desired chain.
func RunMetaStrategies(chainID uint64, wg *sync.WaitGroup) {
	isDone := false
	for {
		FetchStrategiesFromMeta(chainID)
		if !isDone {
			isDone = true
			logs.Info("Meta strategies API is ready")
			wg.Done()
		}
		time.Sleep(1 * time.Minute)
	}
}
