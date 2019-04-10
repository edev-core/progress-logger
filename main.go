package main

import (
	"fmt"
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
	EVENTS_BUCKET   = "events"
)

func dbOpen(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(PROJECTS_BUCKET))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(COMMITS_BUCKET))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(EVENTS_BUCKET))
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

func dbGetEvent(db *bolt.DB, eventId *uuid.UUID) (*Event, error) {
	event := new(Event)

	err := db.View(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte(EVENTS_BUCKET))

		eventBytes := eventBucket.Get(eventId.Bytes())
		if eventBytes == nil {
			return new(EventNotFound)
		}
		return json.Unmarshal(eventBytes, event)
	})
	if err != nil {
		return nil, err
	}

	return event, nil
}

func initEvent(db *bolt.DB, eventId *uuid.UUID, eventName *string) error {
	return db.Update(func(tx *bolt.Tx) error {
		commitBucket := tx.Bucket([]byte(COMMITS_BUCKET))
		projectBucket := tx.Bucket([]byte(PROJECTS_BUCKET))
		eventBucket := tx.Bucket([]byte(EVENTS_BUCKET))

		event := &Event{
			Id:   *eventId,
			Name: *eventName,
		}
		eventBytes, err := json.Marshal(event)
		if err != nil {
			return err
		}
		err = eventBucket.Put(eventId.Bytes(), eventBytes)
		if err != nil {
			return err
		}

		emptyBytes := []byte("[]")
		err = commitBucket.Put(eventId.Bytes(), emptyBytes)
		if err != nil {
			return err
		}
		return projectBucket.Put(eventId.Bytes(), emptyBytes)
	})
}

func main() {
	r := gin.Default()
	main_key := "42"

	db, err := dbOpen("db/progress-logger.db")
	if err != nil {
		fmt.Println("Error opening DB: ", err)
		return
	}

	r.GET("/api/:eventId/commits", func(c *gin.Context) {
		eventId, err := uuid.FromString(c.Param("eventId"))
		fmt.Println(eventId)
		if err != nil {
			c.AbortWithStatus(400)
			return
		}

		commits, err := dbGetCommits(db, &eventId)
		if _, ok := err.(*EventNotFound); ok {
			c.AbortWithStatus(404)
		} else if err != nil {
			c.AbortWithError(500, err)
			return
		}

		c.JSON(200, gin.H{
			"commits": commits,
		})
	})
	r.POST("/api/create", func(c *gin.Context) {
		key := c.PostForm("key")
		eventName := c.PostForm("name")
		fmt.Println(key)

		if eventName == "" {
			c.AbortWithStatus(400)
			return
		}
		if key != main_key {
			c.AbortWithStatus(403)
			return
		}

		eventId, err := uuid.NewV4()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		err = initEvent(db, &eventId, &eventName)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, gin.H{
			"eventId": eventId,
		})
	})
	r.Run()
}
