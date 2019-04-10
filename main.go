package main

import (
	"github.com/gin-gonic/gin"
	// "gopkg.in/libgit2/git2go.v27"
	//	"net/http"
	"encoding/json"
	"github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
	"time"
)

type Project struct {
	URL     string   `json:"url"`
	Name    string   `json:"name"`
	Authors []string `json:"authors"`
}

type Commit struct {
	Project string    `json:"project"`
	Author  string    `json:"author"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
}

type Event struct {
	Name string    `json:"name"`
	Id   uuid.UUID `json:"id"`
}

type EventNotFound struct {
}

func (err *EventNotFound) Error() string {
	return "no such event"
}

const (
	PROJECTS_BUCKET = "projects"
	COMMITS_BUCKET  = "commits"
)

func dbOpen(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(PROJECTS_BUCKET))
		if err != nil {

		}

		_, err = tx.CreateBucketIfNotExists([]byte(COMMITS_BUCKET))
		return err
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func dbGetProjects(db *bolt.DB, eventId *uuid.UUID) ([]Project, error) {
	var projectList []Project

	err := db.View(func(tx *bolt.Tx) error {
		projectBucket := tx.Bucket([]byte(PROJECTS_BUCKET))

		projectBytes := projectBucket.Get(eventId.Bytes())
		if projectBytes == nil {
			return new(EventNotFound)
		}
		return json.Unmarshal(projectBytes, &projectList)
	})
	if err != nil {
		return nil, err
	}

	return projectList, nil
}

func dbStoreProject(db *bolt.DB, project *Project, eventId *uuid.UUID) error {
	projectList, err := dbGetProjects(db, eventId)
	if err != nil {
		return err
	}

	projectList = append(projectList, *project)
	return db.Update(func(tx *bolt.Tx) error {
		projectBucket := tx.Bucket([]byte(PROJECTS_BUCKET))
		projectBytes, err := json.Marshal(&projectList)
		if err != nil {
			return err
		}
		return projectBucket.Put(eventId.Bytes(), projectBytes)
	})
}

func dbGetCommits(db *bolt.DB, eventId *uuid.UUID) ([]Commit, error) {
	var commitList []Commit

	err := db.View(func(tx *bolt.Tx) error {
		commitBucket := tx.Bucket([]byte(COMMITS_BUCKET))

		commitsBytes := commitBucket.Get(eventId.Bytes())
		if commitsBytes == nil {
			return new(EventNotFound)
		}
		return json.Unmarshal(commitsBytes, &commitList)
	})
	if err != nil {
		return nil, err
	}

	return commitList, nil
}

func dbStoreNewCommit(db *bolt.DB, commit *Commit, eventId *uuid.UUID) error {
	commitList, err := dbGetCommits(db, eventId)
	if err != nil {
		return err
	}
	commitList = append(commitList, *commit)

	return db.Update(func(tx *bolt.Tx) error {
		commitBucket := tx.Bucket([]byte(COMMITS_BUCKET))

		commitBytes, err := json.Marshal(&commitList)
		if err != nil {
			return err
		}
		return commitBucket.Put(eventId.Bytes(), commitBytes)
	})
}

func main() {
	r := gin.Default()

	r.GET("/api/commits", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"commits": "wtf",
		})
	})
	r.Run()
}
