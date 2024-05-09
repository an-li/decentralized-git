package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// This enum represents the type of access a user has for a repo
type UserAccess int

const (
	ReadAccess      UserAccess = 1
	ReadWriteAccess UserAccess = 2
	OwnerAccess     UserAccess = 3
	NoAccess        UserAccess = 4
)

// A struct that contains the required data to keep track about who is responsible
// of another user's access in the repository.
type AccessLog struct {
	Authorizer string    `json:"authorizer"`
	Authorized string    `json:"authorized"`
	Timestamp  time.Time `json:"timestamp"`
	UserAccess `json:"userAccess"`
}

type Repository struct {
	Name         string                `json:"name"`
	Author       string                `json:"author"`
	DirectoryCID string                `json:"directoryCID"`
	CommitHashes map[string]bool       `json:"commitHashes"`
	Access       map[string]UserAccess `json:"access"` // Access control map: user -> [permissions]
	Branches     map[string]Branch     `json:"branches"`
	AccessLogs   []AccessLog           `json:"accessLogs"`
}

// This function takes a json string that represents the marshalling of Repo
// and returns a Repo.
// The returned data is valid and consistent
func UnmarshalRepo(objectString string, createdTime time.Time) (Repository, error) {
	var unmarashaledRepo Repository
	json.Unmarshal([]byte(objectString), &unmarashaledRepo)

	repo, _ := CreateNewRepo(unmarashaledRepo.Name, unmarashaledRepo.Author, unmarashaledRepo.DirectoryCID, nil, unmarashaledRepo.AccessLogs, createdTime)

	for _, branch := range unmarashaledRepo.Branches {
		newBranch, _ := CreateNewBranch(branch.Name, nil)
		repo.AddBranch(newBranch, true)

		logsList := make([]Commit, 0, len(branch.Commits))

		for _, v := range branch.Commits {
			logsList = append(logsList, v)
		}

		// sort the logs with from oldest to newest
		sort.Slice(logsList, func(i, j int) bool {
			return logsList[i].Timestamp.UnixMilli() < logsList[j].Timestamp.UnixMilli()
		})

		for i := 0; i < len(logsList); i++ {
			repo.AddCommit(logsList[i], newBranch.Name, false)
		}
	}

	return repo, nil
}

// updates the repo's name
func (repo *Repository) UpdateRepoName(newName string) {
	repo.Name = newName
}

// returns the current access type of the specified user's userName
func (repo *Repository) GetUserAccess(user string) UserAccess {
	if val, exist := repo.Access[user]; exist {
		return val
	}
	return NoAccess
}

// checks if the mentioned user is authorized to do read for the repository.
// anyone included in the repo can read / write
func (repo *Repository) CanRead(user string) bool {
	if val, exist := repo.Access[user]; exist {
		return val == ReadAccess || val == ReadWriteAccess || val == OwnerAccess
	}
	return false
}

// checks if the mentioned user is authorized to do read/write for the repository.
// Collaborators and owners can edit the repository
func (repo *Repository) CanEdit(user string) bool {
	if val, exist := repo.Access[user]; exist {
		return val == ReadWriteAccess || val == OwnerAccess
	}
	return false
}

// checks if the mentioned user is authorized to authorize or revoke ReadWriteAccess for the repository.
// Only owners can auth
func (repo *Repository) IsOwner(user string) bool {
	if val, exist := repo.Access[user]; exist {
		return val == OwnerAccess
	}
	return false
}

// It does the required data writing work to update a user's
// access type in case, the user access update is valid.
func (repo *Repository) UpdateAccess(authorized string, userAccess UserAccess, authorizer string, timestamp time.Time) bool {

	if repo.IsOwner(authorizer) {
		if val, exist := repo.Access[authorized]; exist {
			if val == userAccess {
				return false
			}
		}

		// A user cannot change their own access
		if authorizer == authorized {
			return false
		}

		var accessLog AccessLog
		accessLog.Authorizer = authorizer
		accessLog.Authorized = authorized
		accessLog.Timestamp = timestamp
		accessLog.UserAccess = userAccess

		repo.AccessLogs = append(repo.AccessLogs, accessLog)
		repo.Access[authorized] = userAccess

		return true
	}

	return false
}

// checks if the provided hash has belonged to one of the repo's branches
func (repo *Repository) CommitExists(commitHash string) bool {
	_, exist := repo.CommitHashes[commitHash]
	return exist
}

// checks if the mentioned branch branchName belongs to this repo.
func (repo *Repository) BranchExists(branchName string) bool {
	_, exist := repo.Branches[branchName]
	return exist
}

// returns a list that contains the names of branches in a project
func (repo *Repository) GetBranches() []string {
	keys := make([]string, 0, len(repo.Branches))
	for k := range repo.Branches {
		keys = append(keys, k)
	}
	return keys
}

func (repo *Repository) AddCommitHash(commit Commit) bool {
	repo.CommitHashes[commit.Hash] = true
	return true
}

func (repo *Repository) AddCommitHashes(commits []Commit) bool {
	for _, commit := range commits {
		repo.CommitHashes[commit.Hash] = true
	}
	return true
}

// checks that all hash parents are hashes in the repository
func (repo *Repository) ValidCommit(commit Commit, branchName string) (bool, error) {

	if repo.BranchExists(branchName) {
		branch := repo.Branches[branchName]
		if len(repo.CommitHashes) == 0 {
			return true, nil
		}
		if valid, _ := branch.ValidCommit(commit); valid {

			allParentsAreHashes := true
			for _, parentHash := range commit.ParentHashes {
				allParentsAreHashes = allParentsAreHashes && repo.CommitExists(parentHash)
				if !allParentsAreHashes {
					break
				}
			}

			if allParentsAreHashes {
				return true, nil
			}
		}
	}

	return false, nil
}

// helper function that is needed to create a new Repo instance
func CreateNewRepo(name string, author string, directoryCID string, branches map[string]Branch, accessLogs []AccessLog, createdTime time.Time) (Repository, error) {
	var repo Repository

	repo.Name = name
	repo.Author = author
	repo.DirectoryCID = directoryCID
	repo.CommitHashes = make(map[string]bool)

	if accessLogs != nil {
		repo.AccessLogs = accessLogs
	} else {
		repo.AccessLogs = make([]AccessLog, 0)
		repo.AccessLogs = append(repo.AccessLogs, AccessLog{repo.Author, repo.Author, createdTime, OwnerAccess})
	}

	repo.Access = make(map[string]UserAccess)
	for _, accessLog := range repo.AccessLogs {
		repo.Access[accessLog.Authorized] = accessLog.UserAccess
	}

	if branches == nil {
		// Will only contain main branch
		repo.Branches = make(map[string]Branch)

		var mainBranch = Branch{"main", make(map[string]Commit)}
		repo.Branches[mainBranch.Name] = mainBranch
		fmt.Println("main branch is created!")
	} else {
		repo.Branches = branches
	}

	fmt.Println("empty repo is created!")

	return repo, nil
}

// Check if branch name is not used before creating a new branch
func (repo *Repository) ValidBranch(branch Branch) (bool, error) {
	return !repo.BranchExists(branch.Name), nil
}

// Adds a new branch to the repo if it creates a new valid state
// a new branch must have a new unique name and it must be consistent
// and builds on the current repo state
func (repo *Repository) AddBranch(branch Branch, setIfExists bool) (bool, error) {
	fmt.Println("Trying to add new branch ")

	if valid, _ := repo.ValidBranch(branch); valid || setIfExists {
		fmt.Println("New branch is valid!")

		repo.Branches[branch.Name] = branch
		for _, log := range branch.Commits {
			repo.AddCommitHash(log)
		}

		return true, nil
	}

	return false, nil
}

// Adds a commit to a branch if it creates a new valid state
func (repo *Repository) AddCommit(commit Commit, branchName string, passValidation bool) (bool, error) {

	if valid, _ := repo.ValidCommit(commit, branchName); valid || passValidation {

		branch := repo.Branches[branchName]
		if done, _ := branch.AddCommit(commit, passValidation); done {
			repo.Branches[branchName] = branch
			repo.AddCommitHash(commit)
			return true, nil
		}

	}

	return false, nil
}

// Adds a list of commits one by one to a branch if it creates a new valid state
func (repo *Repository) AddCommits(commits []Commit, branchName string, passValidation bool) (bool, error) {
	for _, commit := range commits {
		pass, _ := repo.AddCommit(commit, branchName, passValidation)
		if !pass {
			return false, errors.New("Could not add commit " + commit.Hash + "!")
		}
	}

	return true, nil
}

// Updates the name of an existing branch to the repo if it creates a new valid state
// a new branch must have a new unique name and it must be consistent
// and builds on the current repo state
func (repo *Repository) UpdateBranchName(branch Branch, newName string) (bool, error) {
	fmt.Println("Trying to update a branch ")

	if repo.BranchExists(branch.Name) && branch.Name != "main" && newName != "main" {
		branch := repo.Branches[branch.Name]
		if !repo.BranchExists(newName) {
			fmt.Println("New branch is valid!")

			// Delete branch with old name from repo
			delete(repo.Branches, branch.Name)

			// Set new name
			branch.Name = newName

			// Add branch back to repo under new name
			repo.Branches[newName] = branch

			return true, nil
		}
	}

	return false, errors.New("Branch " + branch.Name + " not valid!")
}

// Delete a branch from the repo
func (repo *Repository) DeleteBranch(name string) (bool, error) {
	if repo.BranchExists(name) && name != "main" {
		delete(repo.Branches, name)
		return true, nil
	}

	return false, errors.New("Branch " + name + " does not exist for repo " + repo.Name + "!")
}
