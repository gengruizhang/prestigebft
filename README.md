<h1 style="text-align:center;"> PrestigeBFT: Revolutionizing View Changes in BFT Consensus Algorithms with Reputation Mechanisms </h1>


# Update Heads-up

A major update of the code will be posted in August, which incorporates a more efficient communication service between clients and servers and among servers.

# Run PrestigeBFT
GoLang should have been properly installed with `GOPATH` and `GOROOT`. The GoLang version should be at least `go1.17.6`. In addition, three external packages were used (check out `go.mod`).

    // threshold signatures
    go get go.dedis.ch/kyber
    // logging
    go get github.com/sirupsen/logrus
    // some math packages
    go get gonum.org/v1/gonum/


At this stage, we would like to introduce the key parameters in PrestigeBFT.

`-b int` is the batch size\
`-mq int` is the maximum size of message queues\
`-p int` is the number of packing threads used in one consensus instance\
`-pt int` is the number of computing threads used in view changes

`-n int` is the number of servers in total (system configuration)\
`-th int` is the threshold value for threshold signatures (quorums)\
`-id int` is this server's ID\
`-d int` is the emulated network delay

`-ns bool` enables native storage for storing committed entries in plain text files\
`-r bool` enables log responsiveness\
`-gc bool` enables deleting cache entries that have been committed by consensus\
`-repu bool` enables the reputation engine with computing hash computaional work

`-s int` inital server state: 0 : Leader;  1 : Nominee; 2 : Starlet; 3 : Worker\
`-rp bool` prints real time log on the screen\
`perf` enables peak prformance testing configuration, which disables `rp` and `gc`
