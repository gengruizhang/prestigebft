package main

import (
	"crypto/rsa"
	"flag"
	"github.com/dedis/kyber/pairing/bn256"
	"github.com/dedis/kyber/share"
	"sync"
	"time"
)

const NOP = 3 // number of phases
const MaxClientConnection = 100

const (
	PostPhase = iota
	OrderPhase
	CommitPhase
)

var cmdPhase = []string{"PostPhase", "OrderPhase", "CommitPhase"}
var replyPhase = []string{"PostReply", "OrderReply", "CommitReply"}

type ServerId int
type Phase int
type ViewNumber uint64
type BlockId uint64

const (
	PortExternalListener = iota
	PortInternalListenerPostPhase
	PortInternalListenerOrderPhase
	PortInternalListenerCommitPhase
	PortInternalDialPostPhase
	PortInternalDialOrderPhase
	PortInternalDialCommitPhase
	PortInternalElection
	PortExternalElection
)

var PhaseNameWithIndices = struct {
	sync.RWMutex
	p map[Phase]string
}{p: map[Phase]string{
	PortExternalListener:            "Client Listener",
	PortInternalListenerPostPhase:   "Post Phase TCP Listener",
	PortInternalListenerOrderPhase:  "Order Phase TCP Listener",
	PortInternalListenerCommitPhase: "Commit Phase TCP Listener",
	PortInternalElection:            "Election Phase internal UDP Listener",
	PortExternalElection:            "Election Phase external UDP Listener",
}}

//serverIdAndIpRelation is initialized in loadconf.go
var serverConnRegistry = struct {
	sync.RWMutex
	m map[string]ServerId
}{m: make(map[string]ServerId)}

type transit struct {
	epoch uint64
	state int
}

type ServerInfo struct {
	sync.RWMutex
	Index ServerId
	Ip    string
	Ports map[int]string
}

var blockIdCounter uint64

var relayedRequestRecorder = struct {
	sync.RWMutex
	//map[ClientId]map[Timestamp]
	m map[int]map[uint64]int
}{m: make(map[int]map[uint64]int)}

var ServerList []ServerInfo

//threshold signatures
var ServerSecrets [][]byte

var PublicPoly *share.PubPoly
var PrivatePoly *share.PriPoly

var PrivateShare *share.PriShare
var suite = bn256.NewSuite()

var Quorum int

//RSA signatures
var PrivateKey *rsa.PrivateKey
var PublicKeys []*rsa.PublicKey

//var coordinatorsServerInfo [3]ServerInfo

var coordinatorIdOfPhases = struct {
	//map[phaseName]server_id
	sync.RWMutex
	lookup map[Phase]ServerId
}{lookup: make(map[Phase]ServerId)}

var clientConnections = struct {
	sync.RWMutex
	numOfClients  int
	clientSecrets [][]byte

	// mapping allows finding of client address to clientId
	// It is initialized when loading the client config file
	// It is used when a connection comes
	mapping map[string]int

	//conn takes client address as key
	// This can be done if connection is established
	conn [MaxClientConnection]clientConnDock
}{mapping: make(map[string]int)}

//below parameters initialized through func loadParametersFromCommandLine()
var (
	// parameters related to performance

	BatchSize           int
	LenOfMessageQ       int //= 65536
	NumOfPackers        int
	NumOfPenaltyThreads int

	LogLevel     int
	NumOfServers int
	Threshold    int
	ThisServerID int
	Delay        int
	GC           bool
	ServerState  int
	Report       bool
	PerfTest     bool
	NaiveStorage bool
	Reputation   bool

	clusterConfigPath string
	clientConfigPath  string

	InvestigationInterval time.Duration = 1 * time.Second //the time period of an investigation (randomly chose from a range)
	ReplicationTimeout    time.Duration
	ElectionTimeout       time.Duration = 1 * time.Second
)

func loadCmdParameters() {
	flag.IntVar(&BatchSize, "b", 1, "BatchSize")
	flag.IntVar(&LenOfMessageQ, "mq", 65536, "LenOfMessageQ")

	flag.IntVar(&NumOfPackers, "p", 10, "NumOfPackers")
	flag.IntVar(&NumOfPenaltyThreads, "pt", 10, "NumOfPenaltyThreads")

	flag.IntVar(&NumOfServers, "n", 4, "# of servers")
	//flag.IntVar(&Threshold, "th", 2, "threshold")
	flag.IntVar(&ThisServerID, "id", 0, "serverID")
	flag.IntVar(&Delay, "d", 0, "network delay")

	flag.BoolVar(&NaiveStorage, "ns", false, "naive storage")
	flag.BoolVar(&GC, "gc", false, "garbage duty")
	flag.BoolVar(&Reputation, "r", false, "enable reputation")

	flag.IntVar(&ServerState, "s", LEADER, "0 : Leader | 1 : Nominee | 2 : Starlet | 3 : Worker")
	flag.BoolVar(&Report, "rp", false, "report")
	flag.BoolVar(&PerfTest, "perf", true, "peak performance")

	flag.IntVar(&LogLevel, "log", DebugLevel, "0: PanicLevel |"+
		" 1: FatalLevel |"+
		" 2 : ErrorLevel |"+
		" 3: WarnLevel |"+
		" 4: InfoLevel |"+
		" 5: DebugLevel")

	flag.StringVar(&clusterConfigPath, "scp", "./config/cluster_localhost.conf", "cluster config path")
	flag.StringVar(&clientConfigPath, "ccp", "./config/clients_localhost.config", "client config path")
	flag.Parse()

	Quorum = NumOfServers/3*2 + 1
	//Quorum = NumOfServers
	Threshold = Quorum - 1 //leader does not self-talking
}

const (
	PanicLevel = iota //0
	FatalLevel        //1
	ErrorLevel        //2
	WarnLevel         //3
	InfoLevel         //4
	DebugLevel        //5
	TraceLevel        //6
)
