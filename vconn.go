package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
)

var viewChangeError = errors.New("ViewChangeConnErr")

func (v *viewChangeManager) init() {
	for i, _ := range ServerList {
		port, err := strconv.Atoi(ServerList[i].Ports[PortInternalElection])
		if err != nil {
			log.Errorf("establish election conn failed | err: %v", err)
			return
		}

		serverUDPAddr := &net.UDPAddr{
			IP:   net.ParseIP(ServerList[ThisServerID].Ip),
			Port: port,
		}

		v.allServerUDPAddrs = append(v.allServerUDPAddrs, serverUDPAddr)
	}

	conn, err := net.ListenUDP("udp", v.allServerUDPAddrs[ThisServerID])
	if err != nil {
		log.Errorf("establish election listener failed | err: %v", err)
		return
	}
	v.conn = conn

	go viewChangeHandler()

	fmt.Printf("\u2713 ... view change initialization completed; start receiving to vc messages ...\n")

	go v.receivingViewChangeMessages()
}

func (v *viewChangeManager) receivingViewChangeMessages() {
	for {
		m := make([]byte, 1024)
		n, sender, err := v.conn.ReadFromUDP(m)
		if err != nil {
			log.Errorf("%v | sender: %v | err: %v", viewChangeError, sender, err)
			continue
		}

		var receivedVCRequest ViewChangeRequest

		err = json.Unmarshal(m[:n], &receivedVCRequest)

		if err != nil {
			log.Errorf("%v | sender: %v | err: %v", viewChangeError, sender, err)
			continue
		}

		log.Debugf("===>received msg from %s", sender.String())

		viewChangeMsgQ <- receivedVCRequest
	}
}

func (v *viewChangeManager) broadcastToAll(m interface{}) {

	b, err := serialization(m)
	if err != nil {
		log.Errorf("VC broadcast serialization failured | err: %v", err)
		return
	}

	for i, _ := range v.allServerUDPAddrs {
		if i == ThisServerID {
			continue
		}
		_, err := v.conn.WriteToUDP(b, v.allServerUDPAddrs[i])
		if err != nil {
			log.Errorf("%v | broadcast to Server %d failed | err: %v",
				viewChangeError, i, err)
		}
		log.Debugf("vc msg: Sent to server %d -> %s", i, v.allServerUDPAddrs[i].String())
	}
}

func (v *viewChangeManager) sendBackToSingular(m interface{}, serverId int) {

	if b, err := serialization(m); err == nil {
		_, err := v.conn.WriteToUDP(b, v.allServerUDPAddrs[serverId])
		if err != nil {
			log.Errorf("%v | sendToSingular to Server %d failed | err: %v",
				viewChangeError, serverId, err)
		}
	} else {
		log.Errorf("sendBackToSingular serialization err: %v", err)
	}
}
