package main

import (
	"time"
)

// This structure is modeling the necessary data to be stored for each commit
// including data normally stored through git and data required to get the git
// objects through the IPFS Cluster.
type Commit struct {
	Hash          string            `json:"hash"`
	Author        string            `json:"author"`
	AuthorEmail   string            `json:"authorEmail"`
	Message       string            `json:"message"`
	ParentHashes  []string          `json:"parentHashes"`
	Timestamp     time.Time         `json:"timestamp"`
	StorageHashes map[string]string `json:"storageHashes"`
}

// this is a helper function to initialize a new commit object instance
func CreateNewCommit(message string, author string, email string, hash string, timestamp time.Time, parentHashes []string, storageHashes map[string]string) (Commit, error) {
	var log Commit

	log.Message = message
	log.Author = author
	log.AuthorEmail = email
	log.Hash = hash
	log.ParentHashes = parentHashes
	log.Timestamp = timestamp
	log.StorageHashes = storageHashes

	return log, nil
}
