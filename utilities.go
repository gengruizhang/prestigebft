package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"github.com/dedis/kyber/share"
	"github.com/dedis/kyber/sign/bls"
	kyberOldScalar "go.dedis.ch/kyber"
	"popcorn/penKeyGen"
)

func serialization(m interface{}) ([]byte, error) {

	//return proto.Marshal(m)
	//return []byte(fmt.Sprintf("%v", m)), nil
	return json.Marshal(m)
}

func deserialization(b []byte, m interface{}) error {
	//return proto.Unmarshal(*b, m)
	return json.Unmarshal(b, m)
}

func fetchKeyGen() {
	PublicPoly, PrivateShare = FetchPublicPolyAndPrivateKeyShare()
	//v1, v2 := PublicPoly.Info()
	//fmt.Printf("%v | %v\n", v1, v2)
}

func FetchPublicPolyAndPrivateKeyShare() (*share.PubPoly, *share.PriShare) {

	var prosecutorSecrets []kyberOldScalar.Scalar

	for i := 0; i < Threshold; i++ {
		mySecret := ServerSecrets[i]
		secret := suite.G1().Scalar().SetBytes(mySecret)
		prosecutorSecrets = append(prosecutorSecrets, secret)
	}

	//fmt.Printf("secret: %v\n", prosecutorSecrets)
	PriPoly := share.NewProsecutorPriPoly(suite.G2(), Threshold, prosecutorSecrets)
	PrivatePoly = PriPoly
	//fmt.Println("PriPoly:", PriPoly.String())
	return PriPoly.Commit(suite.G2().Point().Base()), PriPoly.Shares(NumOfServers)[ThisServerID]
}

func getDigest(x []byte) []byte {
	r := sha256.Sum256(x)
	return r[:]
}

func validateMACs(clientID int, rMsg, rMACs []byte) bool {

	key := clientConnections.clientSecrets[clientID]
	var macs = hmac.New(sha256.New, key)
	_, err := macs.Write(rMsg)

	if err != nil {
		log.Errorf("MACs.Write failure | err: %v\n", err)
	}
	verifyingMACs := macs.Sum(nil)

	return hmac.Equal(verifyingMACs, rMACs)
}

func signMsgDigest(tx string) ([]byte, []byte, error) {
	digest := getDigest([]byte(tx))
	sig, err := PenSign(digest[:])
	return digest[:], sig, err
}

func PenSign(msg []byte) ([]byte, error) {
	return penKeyGen.Sign(suite, PrivateShare, msg)
}

func PenVerifyPartially(msg, sig []byte) (int, error) {
	return penKeyGen.Verify(suite, PublicPoly, msg, sig)
}

func PenRecovery(sigShares [][]byte, msg *[]byte) ([]byte, error) {
	sig, err := penKeyGen.Recover(suite, PublicPoly, *msg, sigShares, Threshold, NumOfServers)
	return sig, err
}

func PenVerify(msg, sig []byte) error {
	return bls.Verify(suite, PublicPoly.Commit(), msg, sig)
}

func getHashOfMsg(msg interface{}) ([]byte, error) {
	serializedMsg, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("json marshal failed | err: %v", err)
		return nil, err
	}

	msgHash := sha256.New()
	_, err = msgHash.Write(serializedMsg)
	if err != nil {
		return nil, err
	}

	return msgHash.Sum(nil), nil
}

func cryptoSignMsg(msg interface{}) ([]byte, error) {

	serializedMsg, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("json marshal failed | err: %v", err)
		return nil, err
	}

	msgHash := sha256.New()
	_, err = msgHash.Write(serializedMsg)
	if err != nil {
		return nil, err
	}
	msgHashSum := msgHash.Sum(nil)

	// In order to generate the signature, we provide a random number generator,
	// our private key, the hashing algorithm that we used, and the hash sum
	// of our message
	signature, err := rsa.SignPSS(rand.Reader, PrivateKey, crypto.SHA256, msgHashSum, nil)

	return signature, err
}

func cryptoVerify(publicKey *rsa.PublicKey, msgHashSum, signature []byte) error {
	return rsa.VerifyPSS(publicKey, crypto.SHA256, msgHashSum, signature, nil)
}

/*func prepareInfoSig(msgIndex int64, txHash []byte, phase uint) ([]byte, []byte, error){

	cert := ThresholdCertificate{
		MessageIndex:      msgIndex,
		HashOfTransaction: txHash,
		PhaseSymbol:       phase,
	}

	proof, err := serialization(cert)
	if err != nil {
		return nil, nil, err
	}

	infoHash := getDigest(proof)
	infoSig, err := PenSign(infoHash)
	if err != nil {
		return nil, nil, err
	}

	return infoHash, infoSig, nil
}*/
