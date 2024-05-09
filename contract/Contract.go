package main

import (
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// this struct is the wrapper for the different possible functionalities for managing data in the blockchain.
type Contract struct {
}

// this function is required by the hyperledger API
// it is called on a new contract initialization
func (contract *Contract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	fmt.Println("initializing ledger")
	return shim.Success(nil)
}

// this function is responsible for the various possible calling for this Contract
// whether this calling is an invocation or query
func (contract *Contract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {

	function, args := stub.GetFunctionAndParameters()

	fmt.Println("****************************************\nStarting invocation .. \nfunctionName:\t"+function+"\nargs:\t\n", args)
	defer fmt.Println("Invocation end\n")

	if function == "logIn" {
		return contract.logIn(stub, args)
	} else if function == "logOut" {
		return contract.logOut(stub)
	} else if function == "whoAmI" {
		return contract.whoAmI(stub)
	} else if function == "registerNewUser" {
		return contract.registerNewUser(stub, args)
	} else if function == "changePublicKey" {
		return contract.changePublicKey(stub, args)
	} else if function == "addNewRepo" {
		return contract.addNewRepo(stub, args)
	} else if function == "queryRepo" {
		return contract.queryRepo(stub, args)
	} else if function == "renameRepo" {
		return contract.renameRepo(stub, args)
	} else if function == "deleteRepo" {
		return contract.deleteRepo(stub, args)
	} else if function == "clone" {
		return contract.clone(stub, args)
	} else if function == "addNewBranch" {
		return contract.addNewBranch(stub, args)
	} else if function == "renameBranch" {
		return contract.renameBranch(stub, args)
	} else if function == "deleteBranch" {
		return contract.deleteBranch(stub, args)
	} else if function == "queryBranches" {
		return contract.queryBranches(stub, args)
	} else if function == "queryBranch" {
		return contract.queryBranch(stub, args)
	} else if function == "push" {
		return contract.pushOneCommit(stub, args)
	} else if function == "pushMultiple" {
		return contract.pushMultipleCommits(stub, args)
	} else if function == "pull" {
		return contract.queryBranchCommitsAfter(stub, args)
	} else if function == "checkoutLast" {
		return contract.queryLastBranchCommit(stub, args)
	} else if function == "queryUser" {
		return contract.queryUser(stub, args)
	} else if function == "queryUsers" {
		return contract.queryUsers(stub, args)
	} else if function == "updateRepoUserAccess" {
		return contract.updateRepoUserAccess(stub, args)
	} else if function == "queryRepoUserAccess" {
		return contract.queryRepoUserAccess(stub, args)
	}

	return shim.Error("Invalid Smart Contract function name.")
}

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {

	// Create a new Smart Contract
	err := shim.Start(new(Contract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}
