package kademlia

import (
	"log"
	"os"
	"strconv"
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
		log.Printf("Error parsing BETA from environment, using default 5: %v", err)
		return 5
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
	tReplicateStr := os.Getenv("tReplicate")
	tReplicate, err := strconv.Atoi(tReplicateStr)
	if err != nil {
		log.Printf("Error parsing tReplicate from environment, using default 3600: %v", err)
		return 3600
	}

	return tReplicate
}

func gettRepublish() int {
	tRepublishStr := os.Getenv("tRepublish")
	tRepublish, err := strconv.Atoi(tRepublishStr)
	if err != nil {
		log.Printf("Error parsing tRepublish from environment, using default 86400: %v", err)
		return 86400
	}

	return tRepublish
}

func gettExpire() int {
	tExpireStr := os.Getenv("tExpire")
	tExpire, err := strconv.Atoi(tExpireStr)
	if err != nil {
		log.Printf("Error parsing tExpire from environment, using default 3600: %v", err)
		return 3600 + 86400
	}

	return tExpire
}

var (
	ALPHA      = getAlpha()
	BETA       = getBeta()
	K          = getK()
	TTL        = getTTL()
	tReplicate = gettReplicate()
	tRepublish = gettRepublish()
	tExpire    = gettExpire()
)

func ReloadConfig() {
	ALPHA = getAlpha()
	BETA = getBeta()
	K = getK()
	TTL = getTTL()
	tReplicate = gettReplicate()
	tRepublish = gettRepublish()
	tExpire = gettExpire()

	log.Println("Configuration reloaded from environment variables.")
}
