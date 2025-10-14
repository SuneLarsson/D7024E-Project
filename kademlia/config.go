package kademlia

import (
	"log"
	"os"
	"strconv"
	"sync"
)

func getAlpha() int {
	alphaStr := os.Getenv("ALPHA")
	alpha, err := strconv.Atoi(alphaStr)
	if err != nil {
		log.Printf("Error parsing ALPHA from environment, using default 3: %v", err)
		return 3
	}
	return alpha
}

func getBeta() int {
	betaStr := os.Getenv("BETA")
	beta, err := strconv.Atoi(betaStr)
	if err != nil {
		log.Printf("Error parsing BETA from environment, using default 20: %v", err)
		return 20
	}
	return beta
}

func getK() int {
	kStr := os.Getenv("K")
	k, err := strconv.Atoi(kStr)
	if err != nil {
		log.Printf("Error parsing K from environment, using default 20: %v", err)
		return 20
	}
	return k
}

func getTTL() int {
	ttlStr := os.Getenv("TTL")
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil {
		log.Printf("Error parsing TTL from environment, using default 3600: %v", err)
		return 3600
	}

	return ttl
}

func gettReplicate() int {
	tReplicateStr := os.Getenv("TREPLICATE")
	tReplicate, err := strconv.Atoi(tReplicateStr)
	if err != nil {
		log.Printf("Error parsing tReplicate from environment, using default 3600: %v", err)
		return 1
	}

	return tReplicate
}

func gettRepublish() int {
	tRepublishStr := os.Getenv("TREPUBLISH")
	tRepublish, err := strconv.Atoi(tRepublishStr)
	if err != nil {
		log.Printf("Error parsing tRepublish from environment, using default 86400: %v", err)
		return 24
	}

	return tRepublish
}

func gettExpire() int {
	tExpireStr := os.Getenv("TEXPIRE")
	tExpire, err := strconv.Atoi(tExpireStr)
	if err != nil {
		log.Printf("Error parsing tExpire from environment, using default 3600: %v", err)
		return 25
	}

	return tExpire
}

// Global configuration parameters loaded from environment on package init and
// when ReloadConfig is invoked.
//
// The following environment variables are read (all values are integers):
//   - ALPHA: parallelism for iterative lookups (default 3)
//   - BETA: response threshold for lookup rounds (default 20)
//   - K: bucket size / replication factor (default 20)
//   - TTL: time-to-live for local storage entries, in seconds (default 3600)
//   - TREPLICATE: replication interval, in hours (default 1)
//   - TREPUBLISH: republish/refresh interval for original uploader, in hours (default 24)
//   - TEXPIRE: expiration horizon for stored items on a node, in hours (default 25)
//
// Invalid or missing values fall back to the defaults above and a log message is emitted.
var (
	ALPHA       = getAlpha()
	BETA        = getBeta()
	K           = getK()
	TTL         = getTTL()
	tReplicate  = gettReplicate()
	tRepublish  = gettRepublish()
	tExpire     = gettExpire()
	configMutex sync.Mutex // protects concurrent ReloadConfig and reads of config during reload
)

// ReloadConfig reloads configuration values from environment variables.
//
// This function is safe for concurrent use. It updates the package-level
// configuration variables atomically under a mutex and logs completion.
// Use it to apply config changes at runtime without restarting the process.
func ReloadConfig() {
	configMutex.Lock()
	ALPHA = getAlpha()
	BETA = getBeta()
	K = getK()
	TTL = getTTL()
	tReplicate = gettReplicate()
	tRepublish = gettRepublish()
	tExpire = gettExpire()
	configMutex.Unlock()

	log.Println("Configuration reloaded from environment variables.")
}
