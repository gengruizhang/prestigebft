package main

import (
	"encoding/gob"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func handleCliConn(conn net.Conn) {

	log.Warningf("NEW CLIENT REGISTERED | ClientAddr: %s", conn.RemoteAddr())

	ca := clientConnDock{
		c:   &conn,
		enc: gob.NewEncoder(conn),
		dec: gob.NewDecoder(conn),
	}

	clientConnections.Lock()
	if clientId, ok := clientConnections.mapping[conn.RemoteAddr().String()]; ok {
		clientConnections.conn[clientId] = ca
	} else {
		log.Errorf("unidentified client connection| addr: %s", conn.RemoteAddr().String())
		return
	}
	clientConnections.Unlock()

	for {
		var m ClientProposeEntry

		err := ca.dec.Decode(&m)
		if err != nil {
			if err == io.EOF {
				log.Warnf("EOF | cliAddr: %v", conn.RemoteAddr())
				break
			}
			log.Warnf("Gob Decode error | err: %v | cliAddr: %v", err, conn.RemoteAddr())
			continue
		}

		log.Debugf("%s | <ClientId: %d, TS: %d> client req received | time: %s", cmdPhase[PostPhase], m.ClientId, m.Timestamp, time.Now().UTC().String())

		clientRequestQueue <- &m
	}
}

func asyncRelayClientRequest(coordinatorId ServerId) { // as a worker
	DialogConnRegistry.Lock()
	postPhaseDialogInfo := DialogConnRegistry.conns[PostPhase][coordinatorId]
	DialogConnRegistry.Unlock()

	for {
		m, ok := <-clientRequestQueue

		if !ok {
			log.Warnf("clientRequestQueue closed (relay function of server %d)", ThisServerID)
			return
		}

		valid := validateMACs(m.ClientId, m.Transaction, m.Macs)
		if !valid {
			log.Errorln("validation of txMACs failed")
			return
		}

		log.Debugf("request (ts: %d; c: %d) MAC verified @ %v", m.Timestamp, m.ClientId, time.Now().UTC().String())

		relayEntry := WorkerRelayClientRequest{
			ClientProposeEntry: *m,
			Relay:              true,
		}

		dialConnSendMsgBack(relayEntry, postPhaseDialogInfo.enc, PostPhase)
		log.Debugf("relayEntry sent by server %d", ThisServerID)
	}
}

func asyncPackAndDisseminateBlocks() {
	shuffle := struct {
		sync.RWMutex
		counter int
		entries map[int]Entry
	}{
		counter: 0,
		entries: make(map[int]Entry)}

	for {
		m, ok := <-clientRequestQueue

		if !ok {
			log.Infof("clientRequestQueue closed, quiting leader service (server %d)", ThisServerID)
			return
		}

		//verify each client propose
		valid := validateMACs(m.ClientId, m.Transaction, m.Macs)
		if !valid {
			log.Errorln("validation of txMACs failed")
			return
		}

		incrementLogIndex()

		//pack proposes in a block
		entry := Entry{
			TimeStamp: m.Timestamp,
			ClientId:  m.ClientId,
			Tx:        m.Transaction,
		}

		//no need to lock the map, it works chronologically in each thread
		shuffle.entries[shuffle.counter] = entry

		shuffle.counter++
		if shuffle.counter < BatchSize {
			continue
		}

		//t_before_serialization := time.Now()
		bytesOfBatchedEntries, err := serialization(shuffle.entries)
		if err != nil {
			log.Errorf("serialization failed: %v", err)
			return
		}

		hashOfBatchedEntries := getDigest(bytesOfBatchedEntries)

		log.Debugf("%s | before penSign time: %v", cmdPhase[PostPhase], time.Now().UTC().String())

		leaderPartialSignature, err := PenSign(hashOfBatchedEntries)

		if err != nil {
			log.Errorf("%s| threshold signing failed | err: %v\n", cmdPhase[PostPhase], err)
			return
		}

		log.Debugf("%s | after digest and penSign time: %v", cmdPhase[PostPhase], time.Now().UTC().String())
		//The fist phase does not send entries but their hash as a whole
		postEntry := LeaderPostEntry{
			BlockId:              atomic.AddUint64(&blockIdCounter, 1),
			PartialSignature:     leaderPartialSignature,
			HashOfBatchedEntries: hashOfBatchedEntries,
			//Entries:   shuffle.entries,
		}

		//
		// Possible to get replies faster than executing the following code in memory?
		// The parallelism with I/O cannot be achieved when coordinators are separated
		//
		orderFrag := fragment{
			blockHash:  postEntry.HashOfBatchedEntries,
			entries:    shuffle.entries,
			signatures: [][]byte{},
		}

		commitFrag := fragment{
			blockHash:  postEntry.HashOfBatchedEntries,
			entries:    shuffle.entries,
			signatures: [][]byte{},
		}

		orderCache.Lock()
		orderCache.m[postEntry.BlockId] = &orderFrag

		log.Debugf("orderCache.m[%d]: %v", postEntry.BlockId, orderCache.m[postEntry.BlockId].entries)
		orderCache.Unlock()

		commitCache.Lock()
		commitCache.m[postEntry.BlockId] = &commitFrag
		commitCache.Unlock()

		//clear shuffle variables
		shuffle.counter = 0
		shuffle.entries = make(map[int]Entry)

		broadcast(postEntry, PostPhase)

		//t_after_broadcast := time.Now()
		//latencyMeters(cmdPhase[PostPhase], "phase total time", t_phase_start)
		log.Debugf("new PostEntryBlock broadcast -> b_id: %d | b_hash: %s | sig: %s",
			postEntry.BlockId, hex.EncodeToString(postEntry.HashOfBatchedEntries), hex.EncodeToString(postEntry.PartialSignature))
	}
}

func registerIncomingWorkerServers(sConn *net.Conn, phase int) (ServerId, error) {

	serverConnNav.mu.Lock()

	defer serverConnNav.mu.Unlock()

	defer serverConnRegistry.RUnlock()
	//addr[0] is the ip
	serverConnRegistry.RLock()

	//calculate the incoming connection's IP and Port according to views.
	remoteAddr := strings.Split((*sConn).RemoteAddr().String(), ":")

	remoteIp := remoteAddr[0]
	remoteIncomingPort, err := strconv.Atoi(remoteAddr[1])
	if err != nil {
		log.Errorf("unable to convert incoming string port to int: %s", remoteAddr[1])
		return -1, nil
	}

	remoteRegisteredPort := remoteIncomingPort - int(getView())
	registeredServerAddr := remoteIp + ":" + strconv.Itoa(remoteRegisteredPort)

	if sid, ok := serverConnRegistry.m[registeredServerAddr]; ok {
		serverConnNav.n[phase][sid] = &serverConnDock{
			serverId: sid,
			conn:     sConn,
			enc:      gob.NewEncoder(*sConn),
			dec:      gob.NewDecoder(*sConn),
		}

		log.Warningf("%s | new server registered | Id: %v -> Addr: %v\n",
			cmdPhase[phase], sid, (*sConn).RemoteAddr())
		return sid, nil
	} else {
		return -1, errors.New("incoming connection conf was not loaded :(")
	}
}

func handlePostPhaseServerConn(sConn *net.Conn) { //relay message

	sid, err := registerIncomingWorkerServers(sConn, PostPhase)
	if err != nil {
		log.Errorf("%s | err: %v | incoming conn Addr: %v",
			cmdPhase[PostPhase], err, (*sConn).RemoteAddr())
		return
	}

	for {
		var m WorkerRelayClientRequest

		if err := serverConnNav.n[PostPhase][sid].dec.Decode(&m); err == nil {
			go asyncHandleServerRelayMessage(&m, sid)
		} else if err == io.EOF {
			log.Errorf("%s | server %v closed connection | err: %v",
				"Realy", sid, err)
			break
		} else {
			log.Errorf("%s | gob decode Err: %v | conn with ser: %v | remoteAddr: %v",
				"Realy", err, sid, (*sConn).RemoteAddr())
			continue
		}

	}
}

func handleOrderPhaseServerConn(sConn *net.Conn) {
	// Handle PostReply from servers
	sid, err := registerIncomingWorkerServers(sConn, OrderPhase)
	if err != nil {
		log.Errorf("%s | err: %v | incoming conn Addr: %v",
			cmdPhase[OrderPhase], err, (*sConn).RemoteAddr())
		return
	}

	for {
		var m WorkerPostReply

		if err := serverConnNav.n[OrderPhase][sid].dec.Decode(&m); err == nil {
			go asyncHandleServerPostReply(&m, sid)
		} else if err == io.EOF {
			log.Errorf("%s | server %v closed connection | err: %v",
				cmdPhase[OrderPhase], sid, err)
			break
		} else {
			log.Errorf("%s | gob decode Err: %v | conn with ser: %v | remoteAddr: %v",
				cmdPhase[OrderPhase], err, sid, (*sConn).RemoteAddr())
			continue
		}

		//if &m != nil {
		//	go asyncHandleServerPostReply(&m, sid)
		//} else {
		//	log.Errorf("%s received message is nil", cmdPhase[OrderPhase])
		//}
	}
}

func asyncHandleServerRelayMessage(m *WorkerRelayClientRequest, sid ServerId) {
	log.Infof("received a relay message from ServerId: %d | ClientId: %d | ts: %d",
		sid, m.ClientId, m.Timestamp)

	defer relayedRequestRecorder.Unlock()
	relayedRequestRecorder.Lock()

	if _, ok := relayedRequestRecorder.m[m.ClientId]; !ok {
		log.Infof("first time ClientId: %d relays request", m.ClientId)
		relayedRequestRecorder.m[m.ClientId] = make(map[uint64]int)
	}

	if _, ok := relayedRequestRecorder.m[m.ClientId][m.Timestamp]; ok {
		log.Infof("relay message already processed | ServerId: %d | ClientId: %d | ts: %d",
			sid, m.ClientId, m.Timestamp)
		return
	}

	clientRequestQueue <- &m.ClientProposeEntry

	relayedRequestRecorder.m[m.ClientId][m.Timestamp] = 1
}

func asyncHandleServerPostReply(m *WorkerPostReply, sid ServerId) {

	log.Debugf("WorkerPostReply from Server %d| blockId: %d | Sig: %s | time: %s",
		sid, m.BlockId, hex.EncodeToString(m.SelfSig), time.Now().UTC().String())

	log.Debugf("%s| (BlockId:%d; ServerId:%d) WorkerPostReply received @ %v", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())

	//
	// In Golang, map does not support concurrent operations.
	// blockCache, which stores all the blocks with a key of blockId,
	// needs to be released as soon as possible if it needs to be locked.

	orderCache.RLock()
	blockOrderFrag, ok := orderCache.m[m.BlockId]
	orderCache.RUnlock()
	log.Debugf("%s | cache info: b_id: %d | b_info:%v", cmdPhase[OrderPhase], m.BlockId, blockOrderFrag.entries)
	if !ok {
		log.Debugf("%s | no info in cache; consensus may already reached", cmdPhase[OrderPhase])
		return
	}

	//log.Infof("%s | Server %d| order cache fetched | time: %s",
	//	cmdPhase[OrderPhase], sid, time.Now().UTC().String())

	log.Debugf("%s| (BlockId:%d; ServerId:%d) before PenVerifyPartially @ %v", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())

	if !PerfTest {
		serverId, err := PenVerifyPartially(blockOrderFrag.blockHash, m.SelfSig)
		if err != nil {
			log.Errorf("%s | PenVerifyPartially server id: %d | err: %v",
				cmdPhase[OrderPhase], serverId, err)
		}

		log.Debugf("%s| (BlockId:%d; ServerId:%d) after PenVerifyPartially @ %v", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())
	}

	blockOrderFrag.Lock()
	orderedIndicator := len(blockOrderFrag.signatures)

	if orderedIndicator == Quorum {
		log.Errorf("%s | Block has been ordered | logIndicator: %v | Quorum: %v",
			cmdPhase[OrderPhase], orderedIndicator, Quorum)
		blockOrderFrag.Unlock()
		return
	}

	orderedIndicator++
	aggregatedSigShares := append(blockOrderFrag.signatures, m.SelfSig)
	blockOrderFrag.signatures = aggregatedSigShares //update orderedThreshSig

	blockOrderFrag.Unlock()

	if orderedIndicator < Quorum {
		log.Debugf("%s | insufficient votes for ordering | blockId: %v | orderedIndicator: %v",
			cmdPhase[OrderPhase], m.BlockId, orderedIndicator)
		return
	}

	if orderedIndicator > Quorum {
		log.Debugf("%s | block %v already broadcast for ordering", cmdPhase[OrderPhase], m.BlockId)
		return
	}

	log.Debugf("%s| (BlockId:%d; ServerId:%d) consensus reached; before PenRecovery @ %v", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())

	sigThreshed, err := PenRecovery(aggregatedSigShares, &blockOrderFrag.blockHash)
	if err != nil {
		log.Errorf("%s | PenRecovery failed | len(sigShares): %v | error: %v",
			cmdPhase[OrderPhase], len(aggregatedSigShares), err)
		return
	}

	log.Debugf("%s| (BlockId:%d; ServerId:%d) after PenRecovery @ %v", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())

	orderEntry := LeaderOrderEntry{
		BlockId:   m.BlockId,
		ConcatSig: sigThreshed,
		Entries:   blockOrderFrag.entries,
	}

	broadcast(orderEntry, OrderPhase)

	log.Debugf("%s | new OrderEntry has been broadcast for block -> b_id: %d | b_size: %d | b_info: %v| b_combinedSign: %s",
		cmdPhase[OrderPhase], m.BlockId, len(blockOrderFrag.entries), blockOrderFrag.entries, hex.EncodeToString(sigThreshed))

	commitCache.RLock()
	blockCommitFrag, ok := commitCache.m[m.BlockId]
	commitCache.RUnlock()

	if !ok {
		log.Debugf("THIS SHOULD NEVER HAPPEN | blockCommitFrag for block %d is not in cache", m.BlockId)
		return
	}

	log.Debugf("%s | (BlockId:%d; ServerId:%d) before PenSign | time: %s", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())

	if partialSignature, err := PenSign(blockCommitFrag.blockHash); err == nil {
		blockCommitFrag.Lock()
		blockCommitFrag.signatures = [][]byte{partialSignature}
		blockCommitFrag.Unlock()
	} else {
		log.Errorf("%s | threshold signature err: %v", cmdPhase[OrderPhase], err)
	}

	log.Debugf("%s | (BlockId:%d; ServerId:%d) after PenSign | time: %s", cmdPhase[OrderPhase], m.BlockId, sid, time.Now().UTC().String())
}

var notifyClientsLock sync.RWMutex

func notifyClients(blockId uint64, entries *map[int]Entry) {
	defer notifyClientsLock.RUnlock()
	notifyClientsLock.RLock()

	for inBlockTxId, entry := range *entries {
		confirmToClient := ClientConfirm{
			Timestamp:   entry.TimeStamp,
			BlockId:     blockId,
			InBlockTxId: inBlockTxId,
		}

		if err := clientConnections.conn[entry.ClientId].enc.Encode(confirmToClient); err != nil {
			// If the client miss the notification, e.g., connection is down, we do not
			// handle this by keeping the cache.
			//
			// The rational behinds this is that we allow for the failure between server and
			// client, and the failure should not block other tx to be notified with
			// the corresponding deletion of the entire block in cache.
			//
			// The error is only handled within this function, with no error
			// passing to outer functions. It is possible that some transactions failed to
			// be announced to the corresponding clients, but we do not differentiate a
			// partial success from the task. The failures in this phase is common especially
			// clients do not have stable connections, e.g., with a wrongly configured timer.
			//
			log.Errorf("gob encoding to client failed | err: %v ", err)
			continue
		}
		log.Debugf("notify client succeeded | blockId: %v", blockId)
	}
}

func handleCommitPhaseServerConn(sConn *net.Conn) {

	sid, err := registerIncomingWorkerServers(sConn, CommitPhase)

	if err != nil {
		log.Errorf("%s | err: %v | incoming conn Addr: %v",
			cmdPhase[CommitPhase], err, (*sConn).RemoteAddr())
		return
	}

	receiveCounter := int64(0)

	for {
		var m WorkerOrderReply

		err := serverConnNav.n[CommitPhase][sid].dec.Decode(&m)

		counter := atomic.AddInt64(&receiveCounter, 1)

		if err == io.EOF {
			log.Errorf("%v | server %v closed connection | err: %v", time.Now(), sid, err)
			break
		}

		if err != nil {
			log.Errorf("Gob Decode Err: %v | conn with ser: %v | remoteAddr: %v | Now # %v", err, sid, (*sConn).RemoteAddr(), counter)
			continue
		}

		if &m != nil {
			go asyncHandleServerOrderReply(&m, sid)
		} else {
			log.Errorf("received message is nil")
		}
	}
}

func asyncHandleServerOrderReply(m *WorkerOrderReply, sid ServerId) {

	if m == nil {
		log.Errorf("received WorkerOrderReply is empty")
		return
	}
	log.Debugf("%s | (BlockId:%d; ServerId:%d) WorkerOrderReply received | time: %s", cmdPhase[CommitPhase], m.BlockId, sid, time.Now().UTC().String())

	commitCache.RLock()
	commitFragment, ok := commitCache.m[m.BlockId]
	commitCache.RUnlock()

	if !ok {
		log.Debugf("block %d may already committed", m.BlockId)
		return
	}

	if !PerfTest {
		serverId, err := PenVerifyPartially(commitFragment.blockHash, m.SelfSig)
		if err != nil {
			log.Errorf("%s | PenVerifyPartially server id: %d | err: %v", cmdPhase[OrderPhase], serverId, err)
		}
	}

	commitFragment.Lock()
	indicator := len(commitFragment.signatures)

	if indicator == Quorum {
		log.Errorf("This block has been committed| indicator: %v | Quorum: %v", indicator, Quorum)
		commitFragment.Unlock()
		return
	}

	indicator++
	aggregatedSigShares := append(commitFragment.signatures, m.SelfSig)
	commitFragment.signatures = aggregatedSigShares
	commitFragment.Unlock()

	if indicator < Quorum {
		log.Debugf("%s | insufficient votes | blockId: %d | indicator: %d",
			cmdPhase[CommitPhase], m.BlockId, indicator)
		return
	}

	if indicator > Quorum {
		log.Debugf("%s | block %d already broadcast for committing | indicator: %d",
			cmdPhase[CommitPhase], m.BlockId, indicator)
		return
	}

	//now incremented logIndicator == quorum
	log.Debugf(" *** order reply votes suffice *** | block Id: %v | indicator: %d", m.BlockId, indicator)

	log.Debugf("%s | (BlockId:%d; ServerId:%d) consensus reached| before PenRecovery | time: %s", cmdPhase[CommitPhase], m.BlockId, sid, time.Now().UTC().String())

	sigThreshed, err := PenRecovery(aggregatedSigShares, &commitFragment.blockHash)
	if err != nil {
		log.Errorf("%s | PenRecovery failed | len(sigShares): %d | error: %v",
			cmdPhase[CommitPhase], len(aggregatedSigShares), err)
		return
	}
	log.Debugf("%s | (BlockId:%d; ServerId:%d) after PenRecovery | time: %s", cmdPhase[CommitPhase], m.BlockId, sid, time.Now().UTC().String())

	incrementCommitIndex()

	commitEntry := LeaderCommitEntry{
		BlockId:   m.BlockId,
		ConcatSig: sigThreshed,
	}

	broadcast(commitEntry, CommitPhase)
	log.Debugf("%s | (BlockId:%d; ServerId:%d) commitEntry broadcast | time: %s", cmdPhase[CommitPhase], m.BlockId, sid, time.Now().UTC().String())

	notifyClients(m.BlockId, &commitFragment.entries)

	log.Debugf("%s| (BlockId:%d; ServerId:%d) notified client | %s", cmdPhase[CommitPhase], m.BlockId, sid, time.Now().UTC().String())

	// Clear the cache
	// NB: for prototype peak performance evaluation, delete was disabled.
	if !PerfTest {
		return
	}

	txCache.Lock()
	txCache.b[m.BlockId] = TxBlock{
		header:     BlockHeader{},
		v:          0,
		n:          0,
		orderingQC: nil,
		commitQC:   sigThreshed,
		tx:         nil,
	}

	txCache.Unlock()

	orderCache.Lock()
	delete(orderCache.m, m.BlockId)
	log.Debugf("Cache size: %v | Cache of <block Id: %v> was cleared", len(orderCache.m), m.BlockId)
	orderCache.Unlock()

	commitCache.Lock()
	delete(commitCache.m, m.BlockId)
	log.Debugf("Cache size: %v | Cache of <block Id: %v> was cleared", len(commitCache.m), m.BlockId)
	commitCache.Unlock()
}
