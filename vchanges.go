package main

import (
	"net"
	"sync"
	"time"
)

const (
	TypeVoteME = iota
	TypeGrantVote
	TypeNewView
)

var viewChangeMsgQ = make(chan ViewChangeRequest, LenOfMessageQ)

// As a nominee:
var notifyOfAnotherLeaderFound = make(chan int, 1)
var notifyOfSucceededCampaign = make(chan int, 1)
var notifyOfThereIsAnElectionCampaign = make(chan int, 1)

// current design:
// only one conn for all communications.
type viewChangeManager struct {
	sync.RWMutex
	conn              *net.UDPConn
	allServerUDPAddrs []*net.UDPAddr
}

// Below function viewChangeHandler takes the stacked message in the processing queue.
// The message can only be processed chronologically; concurrently processing requires
// additional locks, which were considered but not implemented inside the function.
// The locks are in the statemachine.go file, working similarly as synchronized functions.
func viewChangeHandler() {
	for {
		vcRequest := <-viewChangeMsgQ

		switch vcRequest.MsgType {
		case TypeVoteME:
			go vcManager.validateVoteMeMsgAndSendBackToCandidate(vcRequest.VoteMeMsg)
		case TypeGrantVote:
			go vcManager.validateGrantVoteMsgAndProceedToBeLeader(vcRequest.GrantVoteMsg)
		case TypeNewView:
			go vcManager.validateNewViewMsg(vcRequest.NewViewMsg)
		}
	}
}

func (v *viewChangeManager) startNewElectionCampaignAsStarlet() {
	//myView := incrementView()

	// Should move to a place where starts a campaign
	// Per system design, this should be done through a semaphore; otherwise, there may be too many blocking campaign requests

	notifyOfThereIsAnElectionCampaign <- 1
	setRole(STARLET)

	// do hash computation work here...
	if Reputation {
		// allPastPenalties := getAllPenalties()

		//test case
		allPastPenalties := []int{1, 2, 3, 4, 5}
		t_left := uint64(200)
		t_all := uint64(400)

		v := getView()
		penalty, error := themis.Calculate(allPastPenalties, uint64(v), uint64(v+1), t_left, t_all)
		if error != nil {
			log.Errorf("penalty calculation failed | err: %v", error)
			return
		}
		payPenalty([]byte("mock_latest_txblock"), penalty, 32, NumOfPenaltyThreads)
	}

	// after receiving the result ...
	// put this in separate functions in the future.
	newView := incrementView()
	v.requestVC([]byte("1"), []byte("2"), []byte("3"), newView)
}

// Candidate in view changes
func (v *viewChangeManager) requestVC(latestBlockHash, latestBlockCertificate, foundNonce []byte, viewNumber ViewNumber) {

	setRole(NOMINEE)

	heyGuysVoteMe := CampaignRequest{
		ServerId:               ThisServerID,
		View:                   uint64(viewNumber),
		LatestBlockId:          getCommitIndex(),
		LatestBlockHash:        latestBlockHash,
		LatestBlockCertificate: latestBlockCertificate,
		RepPenaltyScore:        getRepPenaltyScore(),
		FoundNonce:             foundNonce,
		PartialSignature:       nil,
	}

	//constructCommonEndorsement(heyGuysVoteMe)
	// The partial signature signs
	endorse := constructCommonEndorsement(heyGuysVoteMe)
	if partialSig, err := PenSign(endorse); err == nil {
		heyGuysVoteMe.PartialSignature = partialSig
	} else {
		log.Errorf("PenSign failed | err: %v", err)
		return
	}

	vcCache.cacheLock.Lock()

	if _, ok := vcCache.cache[viewNumber]; ok {
		log.Errorf("already requested ...?")
		return
	}

	vcCache.cache[getView()] = vcBlockFragment{
		//voted for itself
		voted:                  true,
		votedFor:               ServerId(ThisServerID),
		commonEndorsement:      constructCommonEndorsement(heyGuysVoteMe),
		concatThreshSigOfVotes: [][]byte{heyGuysVoteMe.PartialSignature},
		counter:                1,
	}

	vcCache.cacheLock.Unlock()

	vcMsg := ViewChangeRequest{
		MsgType:      TypeVoteME,
		VoteMeMsg:    heyGuysVoteMe,
		GrantVoteMsg: GrantVote{},
		NewViewMsg:   Inauguration{},
	}

	timer := time.NewTimer(5 * time.Second)

	v.broadcastToAll(vcMsg)
	//log.Debugf("heyGuysVoteMe broadcast | msg: %+v", heyGuysVoteMe)

	select {
	case <-notifyOfAnotherLeaderFound:
		timer.Stop()
		log.Infof("Another leader found! notifyOfAnotherLeaderFound")
		<-notifyOfThereIsAnElectionCampaign

	case <-timer.C:
		// campaign failed; split votes
		<-notifyOfThereIsAnElectionCampaign
		log.Infof("Time's up! %v, startNewElectionCampaignAsStarlet", time.Now().UTC())
		v.startNewElectionCampaignAsStarlet()

	case <-notifyOfSucceededCampaign:
		timer.Stop()
		<-notifyOfThereIsAnElectionCampaign
		log.Infof("Campaign succeeded!")
	}

	log.Debugf("campaign terminated in view: %d|", heyGuysVoteMe.View)
}

// worker in view changes
func (v *viewChangeManager) validateVoteMeMsgAndSendBackToCandidate(m CampaignRequest) {

	log.Debugf("received CampaignRequest from Server %d | msg: %+v", m.ServerId, m)

	//validate nomineeSignature first...
	receivedView := ViewNumber(m.View)
	myView := getView()

	// epoch validation
	if receivedView < myView {
		log.Infof(" reject CampaignRequest message | Server %d's view: %d | myView: %d",
			m.ServerId, m.View, myView)
		return
	}

	//
	// other validation processes...
	//
	//

	newView := setView(m.View)

	defer vcCache.cacheLock.Unlock()
	vcCache.cacheLock.Lock()

	cache, ok := vcCache.cache[myView]
	if ok && cache.voted {
		log.Infof("already voted in (this) view %d", myView)
		return
	}

	endorse := constructCommonEndorsement(m)

	vcCache.cache[myView] = vcBlockFragment{
		voted:             true,
		votedFor:          ServerId(m.ServerId),
		commonEndorsement: endorse,

		// no need for the rest as a voter
		concatThreshSigOfVotes: nil,
		counter:                0,
	}

	if partialSignature, err := PenSign(endorse); err == nil {
		vcMsg := ViewChangeRequest{
			MsgType:   TypeGrantVote,
			VoteMeMsg: CampaignRequest{},
			GrantVoteMsg: GrantVote{
				ServerId:         ThisServerID,
				View:             uint64(newView),
				IVoteU:           true,
				PartialSignature: partialSignature,
			},
			NewViewMsg: Inauguration{},
		}

		v.sendBackToSingular(vcMsg, m.ServerId)
		//log.Debugf("Vote sent back to server %d| vote: %+v", m.ServerId, vcMsg)
	}
}

func (v *viewChangeManager) validateGrantVoteMsgAndProceedToBeLeader(m GrantVote) {
	log.Debugf("received GrantVote from Server %d | msg: %+v", m.ServerId, m)

	myView := getView()
	if m.View < uint64(myView) {
		log.Infof("received view obsolete: %d vs. myView: %d from Server %d", m.View, myView, m.ServerId)
		return
	}

	if !m.IVoteU {
		log.Infof("[werid] Server %d voted against me", m.ServerId)
		return
	}

	defer vcCache.cacheLock.Unlock()
	vcCache.cacheLock.Lock()

	vcCacheFrag, ok := vcCache.cache[myView]
	if !ok {
		log.Errorf("vcCache.cache[%d] hasn't stored", myView)
		return
	}

	voteCount := vcCacheFrag.counter

	log.Debugf("*vote not enough in view %d* | now: %d | need: %d",
		myView, voteCount, Quorum)

	if voteCount >= Quorum {
		log.Errorf("quorum has reached before")
		return
	}

	voteCount++
	vcCacheFrag.counter = voteCount
	vcCacheFrag.concatThreshSigOfVotes = append(vcCacheFrag.concatThreshSigOfVotes, m.PartialSignature)
	vcCache.cache[myView] = vcCacheFrag

	if voteCount == Quorum {
		log.Debugf("vc quorum reached in view %d!", myView)
		//
		cs, err := PenRecovery(vcCacheFrag.concatThreshSigOfVotes, &vcCacheFrag.commonEndorsement)
		if err != nil {
			log.Errorf("PenRecovery failed | err: %v", err)
			return
		}

		//before broadcast, set up TCP listener to be a leader.
		//...
		state.roleChangingAtInauguration <- systemSates{
			view:     myView,
			leaderId: ServerId(ThisServerID),
		}

		log.Infof("Inaugurated in view %d!", myView)
		//
		//
		nv := Inauguration{
			ServerId:          ThisServerID,
			View:              uint64(myView),
			CombinedSignature: cs,
		}

		v.broadcastToAll(ViewChangeRequest{
			MsgType:      TypeNewView,
			VoteMeMsg:    CampaignRequest{},
			GrantVoteMsg: GrantVote{},
			NewViewMsg:   nv,
		})

		log.Debugf("broadcast new view message in view %d", myView)

		notifyOfSucceededCampaign <- 1
	}
}

func (v *viewChangeManager) validateNewViewMsg(m Inauguration) {
	log.Debugf("received Inauguration from Server %d of view %d", m.ServerId, m.View)

	//...validation...
	// Warning:
	// Do NOT SET yourself as leader here!
	log.Infof("Admit leadership of Server %d in view %d", m.ServerId, m.View)

	state.roleChangingAtInauguration <- systemSates{
		view:     ViewNumber(m.View),
		leaderId: ServerId(m.ServerId),
	}
}
