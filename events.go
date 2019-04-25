package main

import (
	"os"
	"path/filepath"
	"time"

	uuid "github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
)

type Commit struct {
	Project string    `json:"project"`
	Author  string    `json:"author"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
}

type Event struct {
	Name     string    `json:"name"`
	Id       uuid.UUID `json:"id"`
	Projects []Project `json:"projects"`
}

type EventRequest struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key" binding:"required"`
}

type EventNotFound struct {
}

func (err *EventNotFound) Error() string {
	return "no such event"
}

const (
	InvalidEventName = 0
	InvalidKey       = 1
)

type InvalidEventRequest struct {
	ErrorType int
}

func (err *InvalidEventRequest) Error() string {
	if err.ErrorType == InvalidEventName {
		return "Invalid name for event"
	} else if err.ErrorType == InvalidKey {
		return "Provided invalid key"
	} else {
		return "Unexpected error"
	}
}

func CreateEvent(db *bolt.DB, eventRequest *EventRequest, mainKey string) (*uuid.UUID, error) {

	// Checking name validity
	if eventRequest.Name == "" {
		return nil, &InvalidEventRequest{
			ErrorType: InvalidEventName,
		}
	}
	if eventRequest.Key != mainKey {
		return nil, &InvalidEventRequest{
			ErrorType: InvalidKey,
		}
	}

	// Creating a UUID
	eventId, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	// Creating event folder
	eventPath := filepath.Join(GIT_PATH, eventId.String())
	err = os.Mkdir(eventPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	err = initEvent(db, &eventId, &eventRequest.Name)
	if err != nil {
		return nil, err
	}
	return &eventId, nil
}

func FetchCommits(db *bolt.DB, page uint32, limit uint32, eventId *uuid.UUID) ([]Commit, error) {
	// Fetching all commits into RAM
	commits, err := dbGetCommits(db, eventId)
	if err != nil {
		return nil, err
	}

	commitCount := uint32(len(commits))
	if page*limit > commitCount {
		return nil, nil
	}

	end := commitCount - (page * limit)
	if (page+1)*limit > commitCount {
		return commits[0:end], nil
	} else {
		return commits[end-limit : end], nil
	}
}
