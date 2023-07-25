<h1 style="text-align:center;"> PrestigeBFT: Revolutionizing View Changes in BFT Consensus Algorithms with Reputation Mechanisms </h1>

# Artifact
To preserve anonymity of the double-blind review process, we have decided not to disclose the popular cloud platform used for evaluation, as it may reveal our identity. The deployment details are closely linked to this platform, and we will submit them to artifact evaluation once the paper is published.

At this stage, we would like to introduce the key parameters in PrestigeBFT.

`-b int` is the batch size\
`-mq int` is the maximum size of message queues\
`-p int` is the number of packing threads used in one consensus instance\
`-pt int` is the number of computing threads used in view changes\

`-n int` is the number of servers in total (system configuration)\
`-th int` is the threshold value for threshold signatures (quorums)\
`-id int` is this server's ID\
`-d int` is the emulated network delay\

`-ns bool` enables native storage for storing committed entries in plain text files\
`-r bool` enables log responsiveness\
`-gc bool` enables deleting cache entries that have been committed by consensus\
`-repu bool` enables the reputation engine with computing hash computaional work\

`-s int` inital server state: 0 : Leader;  1 : Nominee; 2 : Starlet; 3 : Worker\
`-rp bool` prints real time log on the screen\
`perf` enables peak prformance testing configuration, which disables `rp` and `gc`\
