package main

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	LEADER = iota
	NOMINEE
	STARLET
	WORKER
)

var state = struct {
	sync.RWMutex
	logIndex    uint64
	commitIndex uint64

	viewOpLock sync.RWMutex
	view       ViewNumber

	limen    int
	serverId ServerId
	role     int

	reputationPenalty int

	onGoingInvestigation       chan int         // this works as a lock
	roleChangingAtInauguration chan systemSates // this works as a pipe
}{
	onGoingInvestigation:       make(chan int, 1),
	roleChangingAtInauguration: make(chan systemSates, 1),
}

type systemSates struct {
	view     ViewNumber
	leaderId ServerId
}

var replicationTimer = struct {
	sync.RWMutex
	timer map[ViewNumber]*time.Timer
}{}

//func pushInaugurationInfoToReplication(leaderId ServerId){
//	sysStates := systemSates{
//		view:		getView(),
//		leaderId: 	leaderId,
//	}
//	state.roleChangingAtInauguration <- sysStates
//}

func raiseAnLeaderFailureInvestigation() {
	state.onGoingInvestigation <- 1
}

func closeCurrentLeaderFailureInvestigation() {
	<-state.onGoingInvestigation
}

func startInvestigation(cmtIndexAtInvestigation uint64) {
	// This prototype handles leader failure investigations chronologically.
	// A new investigation waits for the prior investigation to finish. Thus,
	// the invoked startInvestigation() function may become stale if this server
	// keeps calling investigation. However, this can be easily extended to handle
	// multiple investigations simultaneously by increasing the channel capacity.

	raiseAnLeaderFailureInvestigation()

	view := getView()
	replicationTimer.timer[view] = time.NewTimer(5 * time.Second)
	// you have a timer here, to wait for count down.

	misunderstanding := make(chan int, 1)

	go func() {
		for {
			if getCommitIndex() > cmtIndexAtInvestigation {
				misunderstanding <- 1
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	//waiting here to see if this is a misunderstanding of leader failure.
	select {
	case <-misunderstanding:
		replicationTimer.timer[view].Stop()
		closeCurrentLeaderFailureInvestigation()
		log.Infof("investigation at cmtIdx: %d is closed. The leader is still good.", cmtIndexAtInvestigation)

	case <-replicationTimer.timer[view].C:
		closeCurrentLeaderFailureInvestigation()
		log.Infof("investigation at cmtIdx: %d confirmed leader failure. Going to campaign elections", cmtIndexAtInvestigation)
		//invokeNewElectionCampaign()
	}
}

func incrementLogIndex() uint64 {
	return atomic.AddUint64(&state.logIndex, 1)
}

func getLogIndex() uint64 {
	return atomic.LoadUint64(&state.logIndex)
}

func incrementCommitIndex() uint64 {
	return atomic.AddUint64(&state.commitIndex, 1)
}

func getCommitIndex() uint64 {
	return atomic.LoadUint64(&state.commitIndex)
}

func incrementView() ViewNumber {
	defer state.viewOpLock.Unlock()
	state.viewOpLock.Lock()

	state.view++
	return state.view
}

func setView(v uint64) ViewNumber {
	defer state.viewOpLock.Unlock()
	state.viewOpLock.Lock()

	state.view = ViewNumber(v)
	return state.view
}

func getView() ViewNumber {
	defer state.viewOpLock.Unlock()
	state.viewOpLock.Lock()

	return state.view
}

func setRole(r int) {
	state.Lock()
	defer state.Unlock()
	state.role = r
}

func getRole() int {
	state.RLock()
	defer state.RUnlock()
	return state.role
}

func getRepPenaltyScore() int {
	return state.reputationPenalty
}

func incrementRepPenaltyScore() {
	state.reputationPenalty++
}
