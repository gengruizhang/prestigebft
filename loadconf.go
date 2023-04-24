package main

import (
	"bufio"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func newParse(numOfServers, numOfClients, ThisServerID int) {
	// parseClusterCrypto(ThisServerID)
	parseClusterConf(numOfServers, ThisServerID)
	parseClientConf(numOfClients)

	fmt.Printf("... All config files fetched and completed ...\n")
}

func parseClusterCrypto(ThisServerID int) {
	var fileRows []string

	c, err := os.Open(fmt.Sprintf("./config/crypto_%d.conf", ThisServerID))

	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(c)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		fileRows = append(fileRows, scanner.Text())
	}

	err = c.Close()
	if err != nil {
		log.Errorf("close crypto.conf failed | err: %v\n", err)
	}

	var prik []byte

	for i := 0; i < len(fileRows); i++ {
		row := strings.Fields(fileRows[i])

		//first line is this server's private key
		if i == 0 {
			prik, err = hex.DecodeString(row[1])
			if err != nil {
				log.Fatalf("Decode private key string failed | err: %v", err)
				return
			}

			err = json.Unmarshal(prik, &PrivateKey)
			if err != nil {
				log.Fatalf("unmarshal private key bytes failed | err: %v", err)
				return
			}
			continue
		}

		pubk, err := hex.DecodeString(row[1])
		if err != nil {
			log.Fatalf("Decode S%d public key string failed | err: %v", i-1, err)
			return
		}

		var publicKey *rsa.PublicKey
		err = json.Unmarshal(pubk, &publicKey)
		if err != nil {
			log.Fatalf("unmarshal S%d public key bytes failed | err: %v", i-1, err)
			return
		}

		PublicKeys = append(PublicKeys, publicKey)
	}

	fmt.Printf("Fetch server crypto keys finished:\n")
	fmt.Printf("Server %d PrivateKey => %s\n", ThisServerID, hex.EncodeToString(prik))
	for i := 0; i < len(PublicKeys); i++ {
		pub, _ := json.Marshal(PublicKeys[i])
		fmt.Printf("Server %d PublicKey -> %s\n", i, hex.EncodeToString(pub))
	}
}

func parseClusterConf(numOfServers, ThisServerID int) {
	var fileRows []string

	s, err := os.Open(clusterConfigPath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(s)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		fileRows = append(fileRows, scanner.Text())
	}

	err = s.Close()
	if err != nil {
		log.Errorf("close fileServer failed | err: %v\n", err)
	}

	//first line is explanation
	if len(fileRows) != numOfServers+1 {
		log.Errorf("Going to panic | fileRows: %v | n: %v", len(fileRows), numOfServers)
		panic(errors.New("number of servers in config file does not match with provided $n$"))
	}

	for i := 0; i < len(fileRows); i++ {
		// Fist line is instructions
		if i == 0 {
			continue
		}

		var singleSL ServerInfo

		row := strings.Split(fileRows[i], " ")

		i, err := strconv.Atoi(row[0])
		if err != nil {
			panic(err)
		}
		singleSL.Index = ServerId(i)

		singleSL.Ip = row[1]

		ServerSecrets = append(ServerSecrets, []byte(row[2]))

		singleSL.Ports = make(map[int]string)

		singleSL.Ports[PortExternalListener] = row[3]
		singleSL.Ports[PortInternalListenerPostPhase] = row[4]
		singleSL.Ports[PortInternalListenerOrderPhase] = row[5]
		singleSL.Ports[PortInternalListenerCommitPhase] = row[6]
		singleSL.Ports[PortInternalDialPostPhase] = row[7]
		singleSL.Ports[PortInternalDialOrderPhase] = row[8]
		singleSL.Ports[PortInternalDialCommitPhase] = row[9]
		singleSL.Ports[PortInternalElection] = row[10]

		serverConnRegistry.Lock()
		serverConnRegistry.m[singleSL.Ip+":"+row[7]] = singleSL.Index
		serverConnRegistry.m[singleSL.Ip+":"+row[8]] = singleSL.Index
		serverConnRegistry.m[singleSL.Ip+":"+row[9]] = singleSL.Index
		serverConnRegistry.Unlock()

		fmt.Printf("fetched server %d config file: %+v\n", singleSL.Index, singleSL)
		ServerList = append(ServerList, singleSL)
	}

}

func parseClientConf(maxCliConns int) {
	c, err := os.Open(clientConfigPath)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(c)
	scanner.Split(bufio.ScanLines)

	var r []string
	for scanner.Scan() {
		r = append(r, scanner.Text())
	}

	err = c.Close()
	if err != nil {
		log.Errorf("Close clients.config failed | err: %v\n", err)
	}

	if len(r) > maxCliConns {
		log.Errorf("Too many clients")
	}

	clientConnections.Lock()
	for clientId, row := range r {
		//row := strings.TrimSpace(row)
		//row := strings.Split(row, " ")
		row := strings.Fields(row)
		clientConnections.clientSecrets = append(clientConnections.clientSecrets, []byte(row[0]))
		clientIp := row[1]
		clientBasePort, err := strconv.Atoi(row[2])

		if err != nil {
			panic(errors.New("unable to convert client base port"))
		}
		expectedClientConnectionPort := clientBasePort + ThisServerID
		expectedClientConnectionAddr := clientIp + ":" + strconv.Itoa(expectedClientConnectionPort)
		clientConnections.mapping[expectedClientConnectionAddr] = clientId
	}
	clientConnections.Unlock()
}
