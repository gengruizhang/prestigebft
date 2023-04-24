package main

import (
	"sync"
)

var txCache = struct {
	sync.RWMutex
	b map[uint64]TxBlock
}{b: make(map[uint64]TxBlock)}

var orderCache = struct {
	//m map[blockId]
	m map[uint64]*fragment
	sync.RWMutex
}{m: make(map[uint64]*fragment)}

var commitCache = struct {
	m map[uint64]*fragment
	sync.RWMutex
}{m: make(map[uint64]*fragment)}

type fragment struct {
	sync.RWMutex

	blockHash []byte
	//Collect postEntry replies and send orderEntry
	//cryptSigOfLeaderEntry	*[]byte //Use cryptSigOfLeaderPostEntry as a base in reply

	entries map[int]Entry

	//The coordinator of the order phase accumulates threshold
	//signatures from others, append it to orderedThreshSig,
	//which is going to be converted to one threshold signature
	//by function PenRecovery.
	signatures [][]byte
}

var vcCache = struct {
	cacheLock sync.RWMutex
	blockLock sync.RWMutex
	cache     map[ViewNumber]vcBlockFragment
	block     map[ViewNumber]vcBlockPermanent
}{
	cache: make(map[ViewNumber]vcBlockFragment),
	block: make(map[ViewNumber]vcBlockPermanent),
}

type vcBlockFragment struct {
	//used by voters during the election
	voted             bool
	votedFor          ServerId
	commonEndorsement []byte

	//used by candidate
	concatThreshSigOfVotes [][]byte
	counter                int
}

func constructCommonEndorsement(voteMeMsg CampaignRequest) []byte {
	// endorse can be changed differently
	// endorse cannot include partial signature!
	endorse := CampaignRequest{
		ServerId:               voteMeMsg.ServerId,
		View:                   voteMeMsg.View,
		LatestBlockId:          voteMeMsg.LatestBlockId,
		LatestBlockHash:        voteMeMsg.LatestBlockHash,
		LatestBlockCertificate: voteMeMsg.LatestBlockCertificate,
		RepPenaltyScore:        voteMeMsg.RepPenaltyScore,
		FoundNonce:             voteMeMsg.FoundNonce,
		PartialSignature:       nil,
	}

	b, err := getHashOfMsg(endorse)
	if err != nil {
		log.Errorf("getHashOfMsg failed | err: %v", err)
		return nil
	}

	return b
}

// vcBlockPermanent is the view-change block, which is updated upon new view is formed
// and keeps track of servers reputation penalty
type vcBlockPermanent struct {
	sync.RWMutex
	leaderId          ServerId
	reputationPenalty map[ServerId]int
	consumedTxBlocks  map[ServerId]int
	newViewMsg        Inauguration
}
