package main

import (
	"encoding/json"
	"errors"
)

// This structure is modeling a branch in the version control system
type Branch struct {
	Name    string            `json:"name"`
	Commits map[string]Commit `json:"commits"`
}

// This function takes a json string that represents the marshalling of Branch
// and returns a RepoBranch.
// The returned data is valid and consistent
func UnmarshalBranch(objectString string) (Branch, error) {
	var repoBranch Branch

	json.Unmarshal([]byte(objectString), &repoBranch)

	return repoBranch, nil
}

// Create a new branch
func CreateNewBranch(name string, commits map[string]Commit) (Branch, error) {
	var repoBranch Branch
	repoBranch.Name = name

	if commits == nil {
		repoBranch.Commits = make(map[string]Commit)
	} else {
		repoBranch.Commits = commits
	}

	return repoBranch, nil
}

// Check if the hash has been added to the branch before.
func (branch *Branch) CommitExists(hashName string) bool {
	_, exist := branch.Commits[hashName]
	return exist
}

// Checks if a commit has at least one parent in the branch.
func (branch *Branch) ValidCommit(commit Commit) (bool, error) {

	if !branch.CommitExists(commit.Hash) {
		if len(branch.Commits) == 0 {
			return true, nil
		}

		for _, hash := range commit.ParentHashes {
			if branch.CommitExists(hash) {
				timeProgressing := branch.Commits[hash].Timestamp.UnixMilli() < commit.Timestamp.UnixMilli()
				return timeProgressing, nil
			} else {
				return false, errors.New("Previous commit " + hash + " not valid for branch " + branch.Name + "!")
			}
		}
	}

	return false, errors.New("Commit " + commit.Hash + " not valid for branch " + branch.Name + "!")
}

// Adds a Commit to the branch if it can be added according to the info avaiable to the branch.
func (branch *Branch) AddCommit(commitLog Commit, passValidation bool) (bool, error) {

	if valid, _ := branch.ValidCommit(commitLog); valid || passValidation {
		branch.Commits[commitLog.Hash] = commitLog
		return true, nil
	}

	return false, nil
}
