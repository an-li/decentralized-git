package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

func applyPair(stub shim.ChaincodeStubInterface, pair LedgerPair) bool {
	fmt.Println("Key:\t" + string(pair.key))
	fmt.Println("Value:\t" + string(pair.value))

	stub.PutState(pair.key, pair.value)
	return true
}

func deletePair(stub shim.ChaincodeStubInterface, pair LedgerPair) bool {
	fmt.Println("Key:\t" + string(pair.key))

	stub.DelState(pair.key)
	return true
}

func applyPairs(stub shim.ChaincodeStubInterface, pairs []LedgerPair) bool {
	for ind, pair := range pairs {
		fmt.Println("Adding index:\t", ind)
		applyPair(stub, pair)
	}
	return true
}

func deletePairs(stub shim.ChaincodeStubInterface, pairs []LedgerPair) bool {
	for ind, pair := range pairs {
		fmt.Println("Adding index:\t", ind)
		deletePair(stub, pair)
	}
	return true
}

func (contract *Contract) registerNewUser(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// userName, userEmail, publicKey
	// Check if user already exists
	_, failMessage := contract.getUserPublicInfo(stub, args[0])
	if failMessage.Message == "" {
		return shim.Error("User " + args[0] + " already exists!")
	}

	user := User{PublicInfo: UserPublicInfo{args[0], args[1], args[2]}}

	userPair, _ := generateUserDBPair(stub, user)
	applyPair(stub, userPair)

	return shim.Success([]byte("User has successfully been created!"))
}

func (contract *Contract) logIn(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// userName, privateKey
	_, err := contract.getLoggedInUser(stub)
	if err == nil {
		return shim.Error("Another user is currently logged in. Please log out before trying to log in!")
	}

	// userName, publicKey
	// Check if user already exists
	userPublicInfo, failMessage := contract.getUserPublicInfo(stub, args[0])
	if failMessage.Message != "" {
		return shim.Error("User " + args[0] + " does not exist!")
	}
	if userPublicInfo.PublicKey != args[1] {
		return shim.Error("Wrong public key provided for user " + args[0] + "!")
	}

	marshaledUserPublicInfo, _ := json.Marshal(userPublicInfo)

	// Store login info in state DB
	stub.PutState("loggedInUser", marshaledUserPublicInfo)

	return shim.Success([]byte("User " + userPublicInfo.Name + " has successfully logged in"))
}

func (contract *Contract) logOut(stub shim.ChaincodeStubInterface) peer.Response {
	// Store login info in state DB
	stub.DelState("loggedInUser")

	return shim.Success([]byte("Logout successful!"))
}

func (contract *Contract) changePublicKey(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// publicKey

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	user := User{PublicInfo: UserPublicInfo{loggedInUser.Name, loggedInUser.Email, args[0]}}

	userPair, _ := generateUserDBPair(stub, user)
	applyPair(stub, userPair)

	marshaledUserPublicInfo, _ := json.Marshal(user.PublicInfo)

	// Store login info in state DB
	stub.PutState("loggedInUser", marshaledUserPublicInfo)

	return shim.Success([]byte("Public key changed for user " + loggedInUser.Name))
}

func (contract *Contract) addNewRepo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repo

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	currentTime, _ := stub.GetTxTimestamp()

	repo, err := UnmarshalRepo(args[0], currentTime.AsTime())
	if err != nil {
		return shim.Error("Repo is invalid!")
	}

	// checking that the creator is whom they claim to be
	if loggedInUser.Name != repo.Author {
		return shim.Error("Repo creator is not the signing user")
	}

	// check if repo already exists
	repoArgsList := make([]string, 2)
	repoArgsList[0] = repo.Author
	repoArgsList[1] = repo.Name
	_, err = contract.getRepoInstance(stub, repoArgsList)
	if err == nil {
		return shim.Error("Repo already exists")
	}

	repoPairs, _ := generateRepoDBPair(stub, repo)
	applyPairs(stub, repoPairs)

	accessPairs, _ := generateRepoUserAccessesDBPair(stub, repo)
	applyPairs(stub, accessPairs)

	branchPairs, _ := generateRepoBranchesDBPair(stub, repo)
	applyPairs(stub, branchPairs)

	branchCommitPairs, _ := generateRepoBranchesCommitsDBPair(stub, repo)
	applyPairs(stub, branchCommitPairs)

	return shim.Success([]byte("The repo has been added successfully to the blockchain."))
}

func (contract *Contract) renameRepo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, newRepoName

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	if !repo.IsOwner(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " is not authorized to rename this repo")
	}

	contract.deleteRepo(stub, []string{repo.Author, repo.Name})

	repo.Name = args[2]
	marshaledRepo, err := json.Marshal(repo)

	contract.addNewRepo(stub, []string{string(marshaledRepo)})

	return shim.Success([]byte("The repo has been successfully renamed from the blockchain."))
}

func (contract *Contract) deleteRepo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	if !repo.IsOwner(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " is not authorized to delete this repo")
	}

	// Delete commits, branches, access, then repo in this order
	branchCommitPairs, _ := generateRepoBranchesCommitsDBPair(stub, repo)
	deletePairs(stub, branchCommitPairs)

	branchPairs, _ := generateRepoBranchesDBPair(stub, repo)
	deletePairs(stub, branchPairs)

	accessPairs, _ := generateRepoUserAccessesDBPair(stub, repo)
	deletePairs(stub, accessPairs)

	repoPairs, _ := generateRepoDBPair(stub, repo)
	deletePairs(stub, repoPairs)

	return shim.Success([]byte("The repo has been deleted successfully from the blockchain."))
}

func (contract *Contract) addNewBranch(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchBinary

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	repoBranch, err := UnmarshalBranch(args[2])
	if err != nil {
		return shim.Error("RepoBranch is invalid!")
	}

	// generate Repo & check validation
	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	// check authorization
	isAuthorized := repo.CanEdit(loggedInUser.Name)
	if !isAuthorized {
		return shim.Error("User is not authorized to edit this repo")
	}

	valid, err := repo.ValidBranch(repoBranch)
	if err != nil || !valid {
		return shim.Error("RepoBranch could not be added!")
	}

	branchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], repoBranch)
	applyPair(stub, branchPair)

	commitsPairs, _ := generateRepoBranchesCommitsDBPairUsingBranch(stub, args[0], args[1], repoBranch)
	applyPairs(stub, commitsPairs)

	return shim.Success([]byte("The branch has been added successfully to its corresponding repo!"))
}

func (contract *Contract) renameBranch(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName, newBranchName

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	// generate Repo & check validation
	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	// check authorization
	isAuthorized := repo.CanEdit(loggedInUser.Name)
	if !isAuthorized {
		return shim.Error("User is not authorized to edit this repo")
	}

	if !repo.BranchExists(args[2]) {
		return shim.Error("Requested branch does not exist in the repo")
	}

	branch := repo.Branches[args[2]]

	_, err = repo.UpdateBranchName(branch, args[3])
	if err != nil {
		return shim.Error("Unable to rename branch!")
	}

	// Delete branch pairs
	commitsPairs, _ := generateRepoBranchesCommitsDBPairUsingBranch(stub, args[0], args[1], branch)
	deletePairs(stub, commitsPairs)

	branchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], branch)
	deletePair(stub, branchPair)

	newBranch := repo.Branches[args[3]]
	// Add pairs for branch with new name
	newBranchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], newBranch)
	applyPair(stub, newBranchPair)

	newCommitsPairs, _ := generateRepoBranchesCommitsDBPairUsingBranch(stub, args[0], args[1], newBranch)
	applyPairs(stub, newCommitsPairs)

	return shim.Success([]byte("The branch has been renamed in its corresponding repo!"))
}

func (contract *Contract) deleteBranch(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	// generate Repo & check validation
	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	// check authorization
	isAuthorized := repo.CanEdit(loggedInUser.Name)
	if !isAuthorized {
		return shim.Error("User is not authorized to edit this repo")
	}

	if args[2] == "main" {
		return shim.Error("main branch cannot be deleted!")
	}

	if !repo.BranchExists(args[2]) {
		return shim.Error("Requested branch does not exist in the repo")
	}

	branch := repo.Branches[args[2]]

	deleted, err := repo.DeleteBranch(branch.Name)
	if !deleted || err != nil {
		return shim.Error("Could not delete branch " + branch.Name)
	}

	// Delete commits
	commitsPairs, _ := generateRepoBranchesCommitsDBPairUsingBranch(stub, args[0], args[1], branch)
	deletePairs(stub, commitsPairs)

	// Delete branch
	branchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], branch)
	deletePair(stub, branchPair)

	return shim.Success([]byte("The branch has been deleted from its corresponding repo!"))
}

func (contract *Contract) pushOneCommit(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName, commitBinary

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	var commit Commit
	err = json.Unmarshal([]byte(args[3]), &commit)
	if err != nil {
		shim.Error("Could not unmarshal commit!")
	}

	// generate Repo & check validation
	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	// check authorization
	isAuthorized := repo.CanEdit(loggedInUser.Name)
	if !isAuthorized {
		return shim.Error("User is not authorized to edit this repo")
	}

	branchDidNotExist := !repo.BranchExists(args[2])
	var newBranch Branch
	if branchDidNotExist {
		newBranch, _ := CreateNewBranch(args[2], nil)
		branchDidNotExist, _ = repo.AddBranch(newBranch, false)
	} else {
		newBranch = repo.Branches[args[2]]
	}

	valid, err := repo.AddCommit(commit, newBranch.Name, false)
	if err != nil || !valid {
		return shim.Error("Commit could not be added!")
	}

	// Set the branch with the commit added to the repo
	repo.Branches[newBranch.Name] = newBranch

	repoPairs, _ := generateRepoDBPair(stub, repo)
	applyPairs(stub, repoPairs)

	if branchDidNotExist {
		branchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], repo.Branches[newBranch.Name])
		applyPair(stub, branchPair)
	}

	push := Push{newBranch.Name, []Commit{commit}}

	commitsPairs, _ := generateRepoBranchesCommitsDBPairUsingPush(stub, args[0], args[1], push)
	applyPairs(stub, commitsPairs)

	return shim.Success([]byte("The commits have been added successfully to the blockchain"))
}

func (contract *Contract) pushMultipleCommits(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName, listCommits

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	var commitsToAdd []Commit
	err = json.Unmarshal([]byte(args[3]), &commitsToAdd)
	if err != nil {
		return shim.Error("Push is invalid!")
	}

	// generate Repo & check validation
	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	// check authorization
	isAuthorized := repo.CanEdit(loggedInUser.Name)
	if !isAuthorized {
		return shim.Error("User is not authorized to edit this repo")
	}

	if len(commitsToAdd) < 1 {
		return shim.Error("Could not find any commits")
	}

	branchDidNotExist := !repo.BranchExists(args[2])
	var newBranch Branch
	if branchDidNotExist {
		newBranch, _ := CreateNewBranch(args[2], nil)
		branchDidNotExist, _ = repo.AddBranch(newBranch, false)
	} else {
		newBranch = repo.Branches[args[2]]
	}

	valid, err := repo.AddCommits(commitsToAdd, newBranch.Name, false)
	if err != nil || !valid {
		return shim.Error("Commits could not be added! " + err.Error())
	}

	// Set the branch with the commits added to the repo
	repo.Branches[newBranch.Name] = newBranch

	repoPairs, _ := generateRepoDBPair(stub, repo)
	applyPairs(stub, repoPairs)

	if branchDidNotExist {
		branchPair, _ := generateRepoBranchDBPair(stub, args[0], args[1], repo.Branches[newBranch.Name])
		applyPair(stub, branchPair)
	}

	push := Push{newBranch.Name, commitsToAdd}

	commitsPairs, _ := generateRepoBranchesCommitsDBPairUsingPush(stub, args[0], args[1], push)
	applyPairs(stub, commitsPairs)

	return shim.Success([]byte("The commits have been added successfully to the blockchain"))
}

func (contract *Contract) updateRepoUserAccess(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, authorized, userAccess

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	access, err := strconv.Atoi(args[3])
	if err != nil {
		return shim.Error("could not parse access")
	}

	accessTimestamp, err := stub.GetTxTimestamp()

	if repo.UpdateAccess(args[2], UserAccess(access), loggedInUser.Name, accessTimestamp.AsTime()) {
		repoPairs, _ := generateRepoDBPair(stub, repo)
		applyPairs(stub, repoPairs)
		pair, _ := generateRepoUserAccessDBPair(stub, args[0], args[1], args[2], args[3], loggedInUser.Name, accessTimestamp.AsTime())
		applyPair(stub, pair)

		return shim.Success([]byte("Access to the repo has been updated successfully!"))
	}

	return shim.Error("UserAccess was not set! Your access type does not permit you to do the required task")
}
