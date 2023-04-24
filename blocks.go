package main

type serverId int

type BlockHeader struct {
	prevBlockAddress [32]byte
	thisBlockAddress [32]byte
}

type Transactions struct {
	tx   [][]byte
	stat []bool
}

type TxBlock struct {
	header BlockHeader
	v      uint64

	n          uint64
	orderingQC []byte
	commitQC   []byte

	tx map[uint64]Transactions
}

type VcBlock struct {
	header BlockHeader
	v      uint64

	leaderId serverId
	nonce    []byte
	confQC   []byte
	vcQC     []byte

	reputation map[serverId]int
	compIndex  map[serverId]uint64
	repIndex   map[serverId]uint64
}
