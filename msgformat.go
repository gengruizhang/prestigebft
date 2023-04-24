package main

type ClientProposeEntry struct {
	ClientId    int
	Timestamp   uint64
	Transaction []byte
	Macs        []byte
}

type LeaderPostEntry struct {
	v                int
	BlockId          uint64
	PartialSignature []byte

	//HashOfBatchedEntries is the common input of the penSign function
	HashOfBatchedEntries []byte
	//Entries map[int]Entry

	//Private:
	hashedForSig []byte
}

type Entry struct {
	TimeStamp uint64
	ClientId  int
	Tx        []byte
}

//Sent to Coordinator 1
type WorkerRelayClientRequest struct {
	v     int
	Relay bool
	ClientProposeEntry
}

// Sent to Coordinator 2
type WorkerPostReply struct {
	v       int
	BlockId uint64
	SelfSig []byte //this is a threshold signature
}

type LeaderOrderEntry struct {
	v         int
	BlockId   uint64
	ConcatSig []byte //this is a combined threshold signature
	Entries   map[int]Entry
}

// Sent to Coordinator 3
type WorkerOrderReply struct {
	v       int
	BlockId uint64
	SelfSig []byte
}

type LeaderCommitEntry struct {
	v         int
	BlockId   uint64
	ConcatSig []byte //this is a combined threshold signature
}

// ClientConfirm is sent to client
type ClientConfirm struct {
	Timestamp   uint64
	BlockId     uint64
	InBlockTxId int
}

// ViewChangeRequest is the only message type for all view-change messages.
// MsgType distinguishes the differences of requests, types are as follows:
// MsgType: 0 --> CampaignRequest, sent from the nominee to all workers;
// MsgType: 1 --> GrantVote, sent from workers to nominee;
// MsgType: 2 --> Inauguration, sent from the new leader (wining nominee) to all

type ViewChangeRequest struct {
	MsgType      int
	VoteMeMsg    CampaignRequest
	GrantVoteMsg GrantVote
	NewViewMsg   Inauguration
}

type CampaignRequest struct {
	ServerId               int
	View                   uint64
	LatestBlockId          uint64
	LatestBlockHash        []byte
	LatestBlockCertificate []byte

	// RepPenaltyScore is the threshold value
	RepPenaltyScore int
	FoundNonce      []byte

	//
	PartialSignature []byte
}

type GrantVote struct {
	ServerId         int
	View             uint64
	IVoteU           bool
	PartialSignature []byte
}

type Inauguration struct {
	ServerId          int
	View              uint64
	CombinedSignature []byte //converted threshold signatures, only one []byte
}
