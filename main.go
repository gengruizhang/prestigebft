package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"prestigebft/repueng"
	"runtime"
	"sync"
	"time"
)

var log = logrus.New()

var vcManager viewChangeManager
var themis repueng.RepEng

func init() {
	//Quorum = NumOfServers/3*2
	//fmt.Println(4/3*2, 16/3*2, 31/3*2, 61/3*2, 100/3*2)
	loadCmdParameters()
	setLogger()
	setRole(ServerState)
	newParse(NumOfServers, MaxClientConnection, ThisServerID)

	vcManager.init()

	initServerConnNavigatorAndDialogConnRegistryConns(NumOfServers)

	fetchKeyGen()

	// Use initialized roles for operation
	state.roleChangingAtInauguration <- systemSates{
		view:     0,
		leaderId: 0,
	}

	fmt.Printf("-------------------------------\n")
	fmt.Printf("|- System set up information -|\n")
	fmt.Printf("|-----------------------------|\n")
	fmt.Printf("| Batchsize\t| %3d\t|\n", BatchSize)
	fmt.Printf("| Server ID\t| %3d\t|\n", ThisServerID)
	fmt.Printf("| Log level\t| %3d\t|\n", LogLevel)
	fmt.Printf("| # of servers\t| %3d\t|\n", NumOfServers)
	fmt.Printf("| Init role\t| %3d\t|\n", ServerState)
	fmt.Printf("| Quorum\t| %3d\t|\n", Quorum)
	fmt.Printf("| CapOfClientReqQueue\t| %3d\t|\n", cap(clientRequestQueue))
	fmt.Printf("| CapOfVCReqQueue\t| %3d\t|\n", cap(viewChangeMsgQ))
	fmt.Printf("-------------------------------\n")
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	//testReputationEngine()
	var wg sync.WaitGroup
	wg.Add(1)

	go start()

	//if ThisServerID == 1 {
	//	go testViewChange()
	//}

	wg.Wait()
}

func latencyMeters(phaseName, info string, pt time.Time) {
	log.Infof("%s: %s | elapsed: %d", phaseName, info, time.Now().Sub(pt).Nanoseconds())
}
