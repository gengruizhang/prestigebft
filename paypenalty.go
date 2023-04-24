package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var nonceFound = false

func nonceGenerator(length int) []byte {
	var randomBytes = make([]byte, length)

	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}

func computationStarter(digestOfLatestTxBlock []byte, penalty, nonceSize int, wg *sync.WaitGroup) {
	rand.Seed(time.Now().UTC().UnixNano())

	defer wg.Done()

	inner_time := time.Now()
	counter := 0

	for {
		if nonceFound {
			break
		}

		counter++
		var nextBloForComputation []byte
		nonce := nonceGenerator(nonceSize)
		nextBloForComputation = append(digestOfLatestTxBlock, nonce...)
		res := sha256.Sum256(nextBloForComputation)
		hash := res[:]

		if validateHashRes(hash, penalty) {
			//fmt.Printf("%s\n", hex.EncodeToString(nonce))
			//fmt.Printf("limen: %d | result: %s | iteration: %d\n", limen, hex.EncodeToString(hash), counter)
			fmt.Println(time.Now().Sub(inner_time).Milliseconds(), "ms")
			nonceFound = true
			break
		}
	}
}

func validateHashRes(hashResult []byte, limen int) bool {

	result := hex.EncodeToString(hashResult)

	prefix := result[:1]
	for i := 0; i < limen; i++ {
		prefix += result[:1]
	}

	return strings.HasPrefix(result, prefix)
}

func payPenalty(digestOfLatestTxBlock []byte, penalty, nonceSize, numberOfGoroutines int) {
	var wg sync.WaitGroup

	for i := 0; i < numberOfGoroutines; i++ {
		wg.Add(1)
		go computationStarter(digestOfLatestTxBlock, penalty, nonceSize, &wg)
	}

	wg.Wait()
}
