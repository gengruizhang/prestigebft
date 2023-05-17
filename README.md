<h1 style="text-align:center;"> PrestigeBFT: Revolutionizing View Changes in BFT Consensus Algorithms with Reputation Mechanisms </h1>

# About the submission
1. We'd like to clarify Figure 12 in the submission. The red lines show the time cost incurred by faulty servers to possess leadership in view changes performing attacks, and the purple lines show the time cost of non-faulty servers to replace a faulty leader in view changes after the attacks. 

2. We'd also like to address a typo. At line 13 in Algo. State-Transition, the message is `ConfVC.Compt`.

We apologize for the typos, and they have been corrected in the extended version of the paper.

# The extended PrestigeBFT paper
Compared to the submission, the extended version includes an appendix with three sections:  
1. Proofs mentioned in the submission including the client interaction, properties of view changes, safety, and liveness. 
2. Selected questions collected during presentations, lectures, and conversions from various groups including ECE/CS students, professors, and distributed system developers. Questions are arranged according to their related sections.
3. Examples of PrestigeBFT's reputation mechanism, including step-by-step calculation of different server behavior and analysis.

The extended paper can be obtained from folder `./paper/prestigebft-extended.pdf` (it may take a minute to render down)  

*Videos of PrestigeBFT has been temporarily made unavailable in order to preserve the double-blind review process.  

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