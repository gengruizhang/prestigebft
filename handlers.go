package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// stopDialConnSignal
// ! Important! Pushing a value in must be done in the new view
var stopDialConnSignal [NOP]chan struct {
	ViewNumber
	bool
}

type clientConnDock struct {
	c   *net.Conn
	enc *gob.Encoder
	dec *gob.Decoder
}

var serverConnNav = struct {
	n  [NOP][]*serverConnDock // Three phases
	mu sync.RWMutex
}{}

func initServerConnNavigatorAndDialogConnRegistryConns(numOfServers int) {

	for i := 0; i < len(stopDialConnSignal); i++ {
		stopDialConnSignal[i] = make(chan struct {
			ViewNumber
			bool
		}, 1)
	}

	for i := 0; i < len(serverConnNav.n); i++ {
		serverConnNav.n[i] = make([]*serverConnDock, numOfServers)
	}

	for i := 0; i < len(DialogConnRegistry.conns); i++ {
		DialogConnRegistry.conns[i] = make(map[ServerId]dialConn)
	}

}

//
// ---------------------------------------------------------------------------------
// | serverConnNav.n[PostPhase] 	---->	[server[1], server[2], ..., server[n]] |
// | serverConnNav.n[OrderPhase]	---->	[server[1], server[2], ..., server[n]] |
// | serverConnNav.n[CommitPhase]	---->	[server[1], server[2], ..., server[n]] |
// ---------------------------------------------------------------------------------
//
//	---------------------------------------------------------------
// | server[i].encoder --> decoder of server[i]'s dial connection |
// | server[i].decoder <-- encoder of server[i]'s dial connection |
//	---------------------------------------------------------------
//

type serverConnDock struct {
	sync.RWMutex
	serverId ServerId
	conn     *net.Conn
	enc      *gob.Encoder
	dec      *gob.Decoder
}

var DialogConnRegistry = struct {
	sync.RWMutex

	//DialogConnRegistry.conns[phaseNumber][coordinatorId]
	conns []map[ServerId]dialConn
}{conns: make([]map[ServerId]dialConn, NOP)}

type dialConn struct {
	sync.RWMutex
	coordId ServerId
	conn    *net.TCPConn
	enc     *gob.Encoder
	dec     *gob.Decoder
}

func broadcast(e interface{}, phase int) {

	for i := 0; i < len(serverConnNav.n[phase]); i++ {
		if ThisServerID == i {
			continue
		}

		if serverConnNav.n[phase][i] == nil {
			log.Debugf("server %v is not registered in phase %v | msg tried to sent %v:", i, phase, e)
			continue
		}

		go func(i int) {
			err := serverConnNav.n[phase][i].enc.Encode(e)
			if err != nil {
				switch err {
				case io.EOF:
					log.Errorf("server %v closed connection | err: %v", serverConnNav.n[phase][i].serverId, err)
					break
				default:
					log.Errorf("sent to server %v failed | err: %v", serverConnNav.n[phase][i].serverId, err)
				}
			}
		}(i)
	}
}

var clientRequestQueue = make(chan *ClientProposeEntry, LenOfMessageQ)

func statusReport() {
	for {
		rc := getCommitIndex()

		time.Sleep(1 * time.Second)

		statusOnTheBeat := fmt.Sprintf("--------------------------------------------------\n"+
			"> LogIndex: %v | CommitIndex: %v | Throughput: %v | lenOfCliReqsQ: %d;\n",
			getLogIndex(), getCommitIndex(), (getCommitIndex()-rc)*uint64(BatchSize), len(clientRequestQueue))

		fmt.Println(statusOnTheBeat)
		log.Infof(statusOnTheBeat)
	}
}

func start() {

	if Report {
		go statusReport()
	}

	go clientCommunication()

	initialLeadership := <-state.roleChangingAtInauguration

	operatingView := initialLeadership.view

	// System operating as the initial roles passed from cmd
	operatingAsInitialRoles(initialLeadership.leaderId)

	// System continues to monitor view changes messages and adjust roles accordingly
	for leadership := range state.roleChangingAtInauguration {

		log.Infof("role changes triggered in handlers | inauguration msg: %+v", leadership)

		if leadership.view < operatingView {
			log.Infof("system states info stale | sysStates.epoch: %d | operatingEpoch: %d",
				leadership.view, operatingView)
			continue
		}

		myCurrentRole := getRole()
		log.Infof("my current role is %d and the leader is %d in view %d",
			myCurrentRole, leadership.leaderId, leadership.view)

		// I am inaugurated as the next leader and now I am already the leader
		if leadership.leaderId == ServerId(ThisServerID) {

			switch myCurrentRole {
			case LEADER:
				// Do nothing ...
				// I am still the leader ...
				log.Errorf("This should not have happened; a leader cannot be augurated twice in a view")
				continue
			case NOMINEE:
				// I am inaugurated as the next leader and now I am the nominee, which means I've won the election
				// ** To become a leader, you have to be a nominee first; otherwise, something went wrong.

				log.Infof("Server %d is the leader now... setting up TCP listener... ", ThisServerID)

				// Kill all dialog connections to the previous leader
				// ...
				DialogConnRegistry.Lock()
				DialogConnRegistry.conns[PostPhase] = nil
				DialogConnRegistry.conns[OrderPhase] = nil
				DialogConnRegistry.conns[CommitPhase] = nil
				DialogConnRegistry.Unlock()

				// Set up server communication as a leader; start accepting new connections
				go runAsLeaderAndAcceptServerConnections(leadership.leaderId)

			default:
				log.Errorf("I'm inaugurated but I'm not leader or nominee; something went wrong | leadership %v:", leadership)
			}
		} else { //leadership.leaderId != ServerId(ThisServerID)
			switch myCurrentRole {
			case LEADER:
				//okay, I need to step down.

				// close my listener;
				// Current design already closed listeners when n-1 connections received.

				//close my connections;
				// # of total connections: (# of phases) * (number of servers -1) // 3*(n - 1)
				// go close_them_all()
				// This can be done in parallel; no need to wait for closure to start dial conn

				// Set up dial connections to new leader
				coordinatorIdOfPhases.Lock()
				coordinatorIdOfPhases.lookup[PostPhase] = leadership.leaderId
				coordinatorIdOfPhases.lookup[OrderPhase] = leadership.leaderId
				coordinatorIdOfPhases.lookup[CommitPhase] = leadership.leaderId
				coordinatorIdOfPhases.Unlock()

				log.Infof("alright I step down in view %d", getView())
				go runAsWorkerStartDialingLeader()

			case NOMINEE:
				// go back to worker, abort the current campaign
				notifyOfAnotherLeaderFound <- 1

				// More things need to be done here
				setRole(WORKER)

			case STARLET:
				// to be added.
				// abort the hash computation; use a channel to do so
			case WORKER:

				// Okay Boss changed. Let's connect to the new boss
				// Set up dial connections to new leader
				coordinatorIdOfPhases.Lock()
				coordinatorIdOfPhases.lookup[PostPhase] = leadership.leaderId
				coordinatorIdOfPhases.lookup[OrderPhase] = leadership.leaderId
				coordinatorIdOfPhases.lookup[CommitPhase] = leadership.leaderId
				coordinatorIdOfPhases.Unlock()

				log.Infof("Okay Boss changed to Server %d in view %d", leadership.leaderId, getView())
				go runAsWorkerStartDialingLeader()
			}
		}

	}
}

func operatingAsInitialRoles(initLeaderId ServerId) {
	if initLeaderId == ServerId(ThisServerID) {
		// Do we need setRole(LEADER)?
		// In order to prevent messing up the role setting, all
		// setRole functions should be done by the election manager!

		// setRole(LEADER)
		//operatingRole = getRole()

		log.Infof("* leadership claimed by itself: server %d *", ThisServerID)

		go runAsLeaderAndAcceptServerConnections(initLeaderId)

	} else {

		//operatingRole = getRole()
		//coordinatorId := initialLeadership.leaderId
		//setCoordinatorInfos(coordinatorId, coordinatorId, coordinatorId)

		coordinatorIdOfPhases.Lock()
		coordinatorIdOfPhases.lookup[PostPhase] = initLeaderId
		coordinatorIdOfPhases.lookup[OrderPhase] = initLeaderId
		coordinatorIdOfPhases.lookup[CommitPhase] = initLeaderId
		coordinatorIdOfPhases.Unlock()

		go runAsWorkerStartDialingLeader()

	}
}
