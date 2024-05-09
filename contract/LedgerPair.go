package main

import (
	"strconv"
	"time"

	b64 "encoding/base64"
	"fmt"

	"crypto/sha256"
	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// This is the resulting key-value pair generated to store some the data
// in the Hyperledger after the maping from complex data structurs is done.
type LedgerPair struct {
	key   string
	value []byte
}

func generateUserDBPair(stub shim.ChaincodeStubInterface, user User) (LedgerPair, error) {

	userHash := getUserKey(user.PublicInfo.Name, user.PublicInfo.PublicKey)

	var pair LedgerPair

	pair.key = userHash

	value := map[string]interface{}{"docName": "user", "userID": userHash, "name": user.PublicInfo.Name,
		"email": user.PublicInfo.Email, "publicKey": user.PublicInfo.PublicKey}

	pair.value, _ = json.Marshal(value)

	return pair, nil
}

func generateRepoDBPair(stub shim.ChaincodeStubInterface, repo Repository) ([]LedgerPair, error) {

	repoHash := getRepoKey(repo.Author, repo.Name)

	list := make([]LedgerPair, 0)
	var pair LedgerPair

	pair.key = repoHash
	accessLogs, _ := json.Marshal(repo.AccessLogs)

	value := map[string]interface{}{"docName": "repo", "repoID": repoHash, "name": repo.Name,
		"author": repo.Author, "directoryCID": repo.DirectoryCID, "accessLogs": string(accessLogs)}

	pair.value, _ = json.Marshal(value)

	list = append(list, pair)

	return list, nil
}

func generateRepoBranchDBPair(stub shim.ChaincodeStubInterface, author string, repoName string, branch Branch) (LedgerPair, error) {

	repoHash := getRepoKey(author, repoName)

	var pair LedgerPair

	indexName := "index-Branch"
	branchIndexKey, _ := stub.CreateCompositeKey(indexName, []string{repoHash, branch.Name})

	fmt.Println("branchIndexKey : " + branchIndexKey)
	pair.key = branchIndexKey

	value := map[string]interface{}{"docName": "branch", "repoID": repoHash, "branchName": branch.Name}
	pair.value, _ = json.Marshal(value)

	return pair, nil
}

func generateRepoBranchesDBPair(stub shim.ChaincodeStubInterface, repo Repository) ([]LedgerPair, error) {

	list := make([]LedgerPair, 0)

	for _, branch := range repo.Branches {
		pair, _ := generateRepoBranchDBPair(stub, repo.Author, repo.Name, branch)
		list = append(list, pair)
	}

	return list, nil
}

func generateRepoBranchCommitDBPair(stub shim.ChaincodeStubInterface, author string, repoName string, branchName string, commit Commit) (LedgerPair, error) {

	repoHash := getRepoKey(author, repoName)

	var pair LedgerPair

	indexName := "index-BranchCommits"
	branchCommitIndexKey, _ := stub.CreateCompositeKey(indexName, []string{repoHash, branchName, commit.Hash})

	pair.key = branchCommitIndexKey

	parentHashes, _ := json.Marshal(commit.ParentHashes)
	storageHashes, _ := json.Marshal(commit.StorageHashes)
	value := map[string]interface{}{"docName": "commit", "repoID": repoHash, "branchName": branchName, "hash": commit.Hash, "message": commit.Message, "author": commit.Author, "authorEmail": commit.AuthorEmail, "timestamp": commit.Timestamp, "parentHashes": string(parentHashes), "storageHashes": string(storageHashes)}
	pair.value, _ = json.Marshal(value)

	return pair, nil
}

func generateRepoBranchesCommitsDBPair(stub shim.ChaincodeStubInterface, repo Repository) ([]LedgerPair, error) {

	list := make([]LedgerPair, 0)
	for _, branch := range repo.Branches {
		for _, log := range branch.Commits {
			pair, _ := generateRepoBranchCommitDBPair(stub, repo.Author, repo.Name, branch.Name, log)
			list = append(list, pair)
		}
	}

	return list, nil
}

func generateRepoBranchesCommitsDBPairUsingBranch(stub shim.ChaincodeStubInterface, author string, repoName string, repoBranch Branch) ([]LedgerPair, error) {

	list := make([]LedgerPair, 0)

	for _, log := range repoBranch.Commits {
		pair, _ := generateRepoBranchCommitDBPair(stub, author, repoName, repoBranch.Name, log)
		list = append(list, pair)
	}

	return list, nil
}

func generateRepoBranchesCommitsDBPairUsingPush(stub shim.ChaincodeStubInterface, author string, repoName string, push Push) ([]LedgerPair, error) {

	list := make([]LedgerPair, 0)

	for _, log := range push.Commits {
		pair, _ := generateRepoBranchCommitDBPair(stub, author, repoName, push.BranchName, log)
		list = append(list, pair)
	}

	return list, nil
}

func getUserKey(name string, publicKey string) string {

	data := map[string]interface{}{"name": name, "publicKey": publicKey}
	js, _ := json.Marshal(data)

	userHash := sha256.New()
	userHash.Write(js)

	sEnc := b64.StdEncoding.EncodeToString(userHash.Sum(nil))
	fmt.Println("User Hash: ", sEnc)

	return sEnc
}

func getRepoKey(author string, repoName string) string {

	data := map[string]interface{}{"name": repoName, "author": author}
	js, _ := json.Marshal(data)

	repoHash := sha256.New()
	repoHash.Write(js)

	sEnc := b64.StdEncoding.EncodeToString(repoHash.Sum(nil))
	fmt.Println("Repo Hash: ", sEnc)

	return sEnc
}

func generateRepoUserAccessDBPair(stub shim.ChaincodeStubInterface, author string, repoName string, authorized string, userAccess string, authorizer string, timestamp time.Time) (LedgerPair, error) {

	repoHash := getRepoKey(author, repoName)

	var pair LedgerPair

	indexName := "index-RepoUserAccess"
	repoUserAccessIndexKey, _ := stub.CreateCompositeKey(indexName, []string{repoHash, authorized, timestamp.Format(time.RFC3339)})

	fmt.Println(indexName + " : \n" + repoUserAccessIndexKey)
	pair.key = repoUserAccessIndexKey

	value := map[string]interface{}{"docName": "userAccess", "repoID": repoHash, "authorized": authorized, "userAccess": userAccess, "authorizer": authorizer, "timestamp": timestamp}
	pair.value, _ = json.Marshal(value)

	return pair, nil
}

func generateRepoUserAccessesDBPair(stub shim.ChaincodeStubInterface, repo Repository) ([]LedgerPair, error) {

	list := make([]LedgerPair, 0)

	for _, userAccess := range repo.AccessLogs {
		pair, _ := generateRepoUserAccessDBPair(stub, repo.Author, repo.Name, userAccess.Authorized, strconv.Itoa(int(userAccess.UserAccess)), userAccess.Authorizer, userAccess.Timestamp)
		list = append(list, pair)
	}

	return list, nil
}
