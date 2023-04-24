package main

import (
	"net"
)

func runAsLeaderAndAcceptServerConnections(leaderId ServerId) {
	// This runs as a leader
	// For now, one leader takes in charge of all three phases

	go acceptPostPhaseConnections(leaderId)
	go acceptOrderPhaseConnections(leaderId)
	go acceptCommitPhaseConnections(leaderId)

	for i := 0; i < NumOfPackers; i++ {
		go asyncPackAndDisseminateBlocks()
	}

}

func clientCommunication() {
	go acceptClientConnections()
}

func acceptClientConnections() {
	clientListenerAddr := ServerList[ThisServerID].Ip + ":" + ServerList[ThisServerID].Ports[PortExternalListener]
	clientConnListener, err := net.Listen("tcp4", clientListenerAddr)

	if err != nil {
		log.Errorln(err)
		return
	}

	log.Infof("Client connection listener is up")

	for {
		if conn, err := clientConnListener.Accept(); err == nil {
			go handleCliConn(conn)
		} else {
			log.Error(err)
		}
	}
}

func closeTCPListener(l *net.Listener, phaseNum Phase) {
	err := (*l).Close()
	if err != nil {
		defer PhaseNameWithIndices.RUnlock()
		PhaseNameWithIndices.RLock()
		log.Errorf("close %s TCP listener failed | err: %v", PhaseNameWithIndices.p[phaseNum], err)
	}
}

func acceptPostPhaseConnections(leaderId ServerId) {
	postPhaseListenerAddress := ServerList[leaderId].Ip + ":" + ServerList[leaderId].Ports[PortInternalListenerPostPhase]
	postPhaseListener, err := net.Listen("tcp4", postPhaseListenerAddress)

	log.Debugf("PostPhase listener is up in view %d", getView())

	defer closeTCPListener(&postPhaseListener, PortInternalListenerPostPhase)

	if err != nil {
		log.Error(err)
		return
	}

	hasInvoked := 0
	for {

		if conn, err := postPhaseListener.Accept(); err == nil {
			go handlePostPhaseServerConn(&conn)
		} else {
			log.Error(err)
		}

		hasInvoked++
		if hasInvoked == NumOfServers-1 {
			log.Infof("All servers connected; ... closing %s Listener: %s",
				cmdPhase[PostPhase], postPhaseListener.Addr().String())
			break
		}
	}
}

func acceptOrderPhaseConnections(leaderId ServerId) {
	orderPhaseListenerAddress := ServerList[leaderId].Ip + ":" + ServerList[leaderId].Ports[PortInternalListenerOrderPhase]
	orderPhaseListener, err := net.Listen("tcp4", orderPhaseListenerAddress)

	defer closeTCPListener(&orderPhaseListener, PortInternalListenerOrderPhase)

	log.Debugf("OrderPhase listener is up in view %d", getView())

	if err != nil {
		log.Error(err)
		return
	}

	hasInvoked := 0
	for {
		if conn, err := orderPhaseListener.Accept(); err == nil {
			go handleOrderPhaseServerConn(&conn)
		} else {
			log.Error(err)
		}

		hasInvoked++
		if hasInvoked == NumOfServers-1 {
			log.Infof("All servers connected; ... closing %s Listener: %v",
				cmdPhase[OrderPhase], orderPhaseListener.Addr().String())
			break
		}
	}
}

func acceptCommitPhaseConnections(leaderId ServerId) {
	commitPhaseListenerAddress := ServerList[leaderId].Ip + ":" + ServerList[leaderId].Ports[PortInternalListenerCommitPhase]
	commitPhaseListener, err := net.Listen("tcp4", commitPhaseListenerAddress)

	defer closeTCPListener(&commitPhaseListener, PortInternalListenerCommitPhase)

	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("CommitPhase listener is up in view %d", getView())

	hasInvoked := 0
	for {

		if conn, err := commitPhaseListener.Accept(); err == nil {
			go handleCommitPhaseServerConn(&conn)
		} else {
			log.Error(err)
		}

		hasInvoked++
		if hasInvoked == NumOfServers-1 {
			log.Infof("All servers connected; ... closing %s Listener: %s",
				cmdPhase[CommitPhase], commitPhaseListener.Addr().String())
			break
		}
	}
}
