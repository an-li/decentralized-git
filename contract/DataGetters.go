package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

func (contract *Contract) getLoggedInUser(stub shim.ChaincodeStubInterface) (UserPublicInfo, error) {
	var loggedInUserInfo UserPublicInfo

	loggedInUserMarshaled, err := stub.GetState("loggedInUser")
	if err != nil {
		return loggedInUserInfo, errors.New("No user is logged in!")
	}

	err = json.Unmarshal(loggedInUserMarshaled, &loggedInUserInfo)
	if err != nil {
		return loggedInUserInfo, errors.New("Invalid connected user information!")
	}

	return loggedInUserInfo, nil
}

func (contract *Contract) whoAmI(stub shim.ChaincodeStubInterface) peer.Response {
	loggedInUserMarshaled, err := stub.GetState("loggedInUser")
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(loggedInUserMarshaled)
}

func (contract *Contract) getRepoInstance(stub shim.ChaincodeStubInterface, args []string) (Repository, error) {
	// repoAuthor, repoName

	fmt.Println("\n---------------------------\nQuerying the ledger .. getRepoInstance", args)
	defer fmt.Println("---------------------------")

	if len(args) < 2 {
		var repo Repository
		fmt.Println("Incorrect number of arguments. Expecting 2")
		return repo, errors.New("Incorrect number of arguments. Expecting 2")
	}

	// getting the required information from first table.
	repoHash := getRepoKey(args[0], args[1])
	repoData, err := stub.GetState(repoHash)
	if err != nil {
		var repo Repository
		fmt.Println("Could not find requested Repo: ", err)
		return repo, errors.New("Could not find requested Repo")
	}
	fmt.Println("Found this repo:", string(repoData))
	// unmarashaling the data
	structuredRepoData := map[string]string{}
	err = json.Unmarshal(repoData, &structuredRepoData)
	if err != nil {
		var repo Repository
		fmt.Println("Could not unmarshal requested repo: ", err)
		return repo, errors.New("Could not unmarashal requested repo")
	}
	users, _ := contract.getRepoUsers(stub, repoHash)

	currentTime, _ := stub.GetTxTimestamp()

	repo, _ := CreateNewRepo(structuredRepoData["name"], structuredRepoData["author"], structuredRepoData["directoryCID"], nil, users, currentTime.AsTime())

	// getting the repo branches
	branchQueryString := fmt.Sprintf("{\"selector\": {\"docName\": \"branch\", \"repoID\": \"%s\"},\"fields\": [\"repoID\", \"branchName\"]}", repoHash)
	branchResultsIterator, err := stub.GetQueryResult(branchQueryString)
	if err != nil {
		fmt.Println("Could not find Requested Branch: ", err)
		var repo Repository
		return repo, err
	}
	defer branchResultsIterator.Close()
	//iterating over branches
	for branchResultsIterator.HasNext() {
		branchString, err := branchResultsIterator.Next()
		if err != nil {
			fmt.Println("Could not proceed to next branch: ", err)
			var repo Repository
			return repo, err
		}

		structuredBranchData := map[string]string{}
		err = json.Unmarshal(branchString.Value, &structuredBranchData)
		fmt.Println("Found This Branch: ", structuredBranchData)
		if err != nil {
			var repo Repository
			fmt.Println("Could not unmarshal requested Branch: ", err)
			return repo, errors.New("Could not unmarashal requested Branch")
		}
		branch, _ := CreateNewBranch(structuredBranchData["branchName"], nil)
		branchAdded, err := repo.AddBranch(branch, true)
		if branchAdded {
			fmt.Println("Branch added to repo: ", branch)
		} else {
			return repo, err
		}

		//adding branch commits
		commitsQueryString := fmt.Sprintf("{\"selector\": {\"docName\": \"commit\", \"repoID\": \"%s\", \"branchName\": \"%s\"},\"fields\": [\"repoID\", \"branchName\", \"hash\", \"message\", \"author\", \"authorEmail\", \"timestamp\", \"parentHashes\", \"storageHashes\"]}", repoHash, branch.Name)
		commitsResultsIterator, err := stub.GetQueryResult(commitsQueryString)
		if err != nil {
			var repo Repository
			fmt.Println("Could not find requested commit: ", err)
			return repo, err
		}
		defer commitsResultsIterator.Close()
		//iterating over branches
		for commitsResultsIterator.HasNext() {
			commitString, err := commitsResultsIterator.Next()
			if err != nil {
				fmt.Println("Could not proceed to requested commit: ", err)
				var repo Repository
				return repo, err
			}

			structuredCommitData := map[string]string{}
			err = json.Unmarshal(commitString.Value, &structuredCommitData)
			fmt.Println("Found This Commit: ", structuredCommitData)
			if err != nil {
				var repo Repository
				fmt.Println("Could not unmarshal requested commit: ", err)
				return repo, errors.New("Could not unmarshal requested commit")
			}
			var ph []string
			_ = json.Unmarshal([]byte(structuredCommitData["parentHashes"]), &ph)
			var sh map[string]string
			_ = json.Unmarshal([]byte(structuredCommitData["storageHashes"]), &sh)

			parsedTimestamp, _ := time.Parse(time.RFC3339Nano, structuredCommitData["timestamp"])

			commit, _ := CreateNewCommit(structuredCommitData["message"], structuredCommitData["author"], structuredCommitData["authorEmail"], structuredCommitData["hash"], parsedTimestamp, ph, sh)
			fmt.Println("and the commit became \t", commit)
			commitAdded, err := repo.AddCommit(commit, branch.Name, true)
			if commitAdded {
				fmt.Println("Commit added to branch: ", commit)
			} else {
				return repo, err
			}
		}

	}

	fmt.Println("This is the final fetched Repo: ", repo)
	return repo, nil
}

func (contract *Contract) queryRepo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName
	_, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. getRepo", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2.")
	}

	repoHash := getRepoKey(args[0], args[1])

	repoData, err := stub.GetState(repoHash)

	if err != nil {
		return shim.Error("Repo does not exist")
	}

	fmt.Println("Found this repo:", string(repoData))

	return shim.Success(repoData)
}

func (contract *Contract) clone(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName
	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. clone", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 3.")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	if !repo.CanRead(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " does not have read access to " + args[1])
	}

	fmt.Println("Found this repo:", repo)

	j, _ := json.Marshal(repo)
	return shim.Success(j)
}

func (contract *Contract) queryBranches(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName
	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. queryBranches", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 3.")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}
	if !repo.CanRead(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " does not have read access to " + args[1])
	}

	fmt.Println("Found these branches:", repo.GetBranches())

	j, _ := json.Marshal(repo.GetBranches())
	return shim.Success(j)
}

func (contract *Contract) queryBranch(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName
	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. queryBranch", args)

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3.")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}
	if !repo.CanRead(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " does not have read access to " + args[1])
	}

	if !repo.BranchExists(args[2]) {
		fmt.Println("Requested Branch Not found")
		return shim.Error("Requested Branch Not found")
	}

	branch := repo.Branches[args[2]]
	fmt.Println("Found these branches:", branch)

	serialized, _ := json.Marshal(branch)
	return shim.Success(serialized)
}

func (contract *Contract) queryBranchCommitsAfter(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName, commitId

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. queryBranchCommitsAfter", args)

	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4.")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	if !repo.CanRead(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " does not have read access to " + args[1])
	}

	if !repo.BranchExists(args[2]) {
		fmt.Println("Requested Branch Not found")
		return shim.Error("Requested Branch Not found")
	}

	branch := repo.Branches[args[2]]
	fmt.Println("Found this branch:", branch)

	commits := make([]Commit, 0)

	// Commit exists?
	t := time.Unix(0, 0)
	if !branch.CommitExists(args[3]) && args[3] != "" {
		fmt.Println("Requested commit Not found")
		return shim.Error("Requested commit Not found")
	}

	if args[3] != "" {
		t = branch.Commits[args[3]].Timestamp
	}

	// get all commits after this time
	for _, log := range branch.Commits {
		if log.Timestamp.UnixNano() > t.UnixNano() {
			commits = append(commits, log)
		}
	}

	serialized, _ := json.Marshal(commits)
	return shim.Success(serialized)
}

func (contract *Contract) queryLastBranchCommit(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName, branchName

	loggedInUser, err := contract.getLoggedInUser(stub)
	if err != nil {
		return shim.Error("Please log in first!")
	}

	fmt.Println("Querying the ledger .. queryBranchCommits", args)

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3.")
	}

	repo, err := contract.getRepoInstance(stub, args)
	if err != nil {
		return shim.Error("Repo does not exist")
	}

	if !repo.CanRead(loggedInUser.Name) {
		return shim.Error("User " + loggedInUser.Name + " does not have read access to " + args[1])
	}

	if !repo.BranchExists(args[2]) {
		fmt.Println("Requested Branch Not found")
		return shim.Error("Requested Branch Not found")
	}

	branch := repo.Branches[args[2]]
	fmt.Println("Found this branch:", branch)

	t := time.Unix(0, 0)
	hash := ""
	// get all commits after this time
	for _, log := range branch.Commits {
		if log.Timestamp.UnixMilli() >= t.UnixMilli() {
			t = log.Timestamp
			hash = log.Hash
		}
	}

	return shim.Success([]byte(hash))
}

func (contract *Contract) getUserPublicInfo(stub shim.ChaincodeStubInterface, userName string) (UserPublicInfo, peer.Response) {
	var userInfo UserPublicInfo

	userQueryString := fmt.Sprintf("{\"selector\": {\"docName\": \"user\", \"name\": \"%s\"},\"fields\": [\"name\", \"email\", \"publicKey\"]}", userName)
	userResultsIterator, err := stub.GetQueryResult(userQueryString)
	if err != nil || !userResultsIterator.HasNext() {
		fmt.Println("Could not find Requested User: ", err)
		return userInfo, shim.Error("User does not exist")
	}
	defer userResultsIterator.Close()

	for userResultsIterator.HasNext() {
		userString, err := userResultsIterator.Next()
		if err != nil {
			fmt.Println("Could not proceed to next user: ", err)
			return userInfo, shim.Error("Could not proceed to next user")
		}

		structuredUserData := map[string]string{}
		err = json.Unmarshal(userString.Value, &structuredUserData)
		fmt.Println("Found This User: \t", structuredUserData)
		if err != nil {
			fmt.Println("Could not unmarshal requested user: ", err)
			return userInfo, shim.Error("Could not unmarshal requested user")
		}

		userInfo.Name = structuredUserData["name"]
		userInfo.Email = structuredUserData["email"]
		userInfo.PublicKey = structuredUserData["publicKey"]
	}

	return userInfo, shim.Success([]byte(""))
}

func (contract *Contract) queryUser(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// userName

	fmt.Println("Querying the ledger .. queryUser", args)

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1.")
	}

	userInfo, failMessage := contract.getUserPublicInfo(stub, args[0])
	if failMessage.Message != "" {
		return failMessage
	}

	serialized, _ := json.Marshal(userInfo)
	return shim.Success(serialized)
}

func (contract *Contract) queryUsers(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// [userName1 userName2]

	fmt.Println("Querying the ledger .. queryUser", args)

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1.")
	}

	userPublicInfos := make([]UserPublicInfo, 0)
	var userNames []string
	_ = json.Unmarshal([]byte(args[0]), &userNames)

	for _, username := range userNames {
		userInfo, failMessage := contract.getUserPublicInfo(stub, username)
		if failMessage.Message != "" {
			continue
		}
		userPublicInfos = append(userPublicInfos, userInfo)
	}

	serialized, _ := json.Marshal(userPublicInfos)
	return shim.Success(serialized)
}

func (contract *Contract) getRepoUsers(stub shim.ChaincodeStubInterface, repoHash string) ([]AccessLog, peer.Response) {

	accessLog := make([]AccessLog, 0)

	accessQueryString := fmt.Sprintf("{\"selector\": {\"docName\": \"userAccess\", \"repoID\": \"%s\"},\"fields\": [\"authorized\", \"userAccess\", \"authorizer\", \"timestamp\"]}, \"sort\": [{\"timestamp\": \"asc\"}],", repoHash)
	accessResultsIterator, err := stub.GetQueryResult(accessQueryString)
	if err != nil {
		fmt.Println("Could not find Repo Access: ", err)
		return accessLog, shim.Error("Repo Access does not exist")
	}
	defer accessResultsIterator.Close()

	for accessResultsIterator.HasNext() {
		accessString, err := accessResultsIterator.Next()
		if err != nil {
			fmt.Println("Could not proceed to user access: ", err)
			return accessLog, shim.Error("Could not proceed to next user access")
		}

		structuredAccessData := map[string]string{}
		err = json.Unmarshal(accessString.Value, &structuredAccessData)
		fmt.Println("Found This User: \t", structuredAccessData)
		if err != nil {
			fmt.Println("Could not unmarshal requested User: ", err)
			return accessLog, shim.Error("Could not unmarshal requested User")
		}
		var parsedTimestamp time.Time
		parsedTimestamp, err = time.Parse(time.RFC3339Nano, structuredAccessData["timestamp"])
		if err != nil {
			fmt.Println("Could not parse timestamp: ", err)
			return accessLog, shim.Error("Could not parse timestamp")
		}
		access, err := strconv.Atoi(structuredAccessData["userAccess"])
		if err != nil {
			fmt.Println("Could not parse UserAccess: ", access, err)
			return accessLog, shim.Error("Could not parse UserAccess")
		} else {
			accessLog = append(accessLog, AccessLog{structuredAccessData["authorizer"], structuredAccessData["authorized"], parsedTimestamp, UserAccess(access)})
		}
	}

	return accessLog, shim.Success([]byte(""))
}

func (contract *Contract) queryRepoUserAccess(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// repoAuthor, repoName

	fmt.Println("Querying the ledger .. queryRepoUserAccess", args)

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2.")
	}

	repoHash := getRepoKey(args[0], args[1])

	users, failMessage := contract.getRepoUsers(stub, repoHash)
	if failMessage.Message != "" {
		return failMessage
	}

	serialized, _ := json.Marshal(users)
	return shim.Success(serialized)
}
