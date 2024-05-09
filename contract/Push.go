package main

import (
	"encoding/json"
)

// This struct is used to model the required data to add a push
// to a repository
type Push struct {
	BranchName string   `json:"branchName"`
	Commits    []Commit `json:"commits"`
}

// Helper function that creates a new object instance of a Push
func CreateNewPush(branchName string, commits []Commit) (Push, error) {
	var log Push
	log.BranchName = branchName
	log.Commits = commits

	return log, nil
}

// This function takes a json string that represents the marshalling of Push
// and returns a Push.
// The returned data is valid and consistent
func UnmarshalPush(objectString string) (Push, error) {
	var log Push

	err := json.Unmarshal([]byte(objectString), &log)

	return log, err
}
