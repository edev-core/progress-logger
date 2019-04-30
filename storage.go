package main

import (
	"encoding/json"
	"github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
)

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

func dbGetProject(db *bolt.DB, projectId *uuid.UUID) (*Project, error) {
	project := new(Project)

	err := db.View(func(tx *bolt.Tx) error {
		projectBucket := tx.Bucket([]byte(PROJECTS_BUCKET))

		projectBytes := projectBucket.Get(projectId.Bytes())
		if projectBytes == nil {
			return new(EventNotFound)
		}
		return json.Unmarshal(projectBytes, project)
	})
	if err != nil {
		return nil, err
	}

	return project, nil
}

func dbStoreProject(db *bolt.DB, project *Project) error {
	return db.Update(func(tx *bolt.Tx) error {
		projectBucket := tx.Bucket([]byte(PROJECTS_BUCKET))

		projectBytes, err := json.Marshal(project)
		if err != nil {
			return err
		}
		return projectBucket.Put(project.Id.Bytes(), projectBytes)
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

func dbAddProjectToEvent(db *bolt.DB, eventId *uuid.UUID, project *Project) error {
	event, err := dbGetEvent(db, eventId)
	if err != nil {
		return err
	}
	event.Projects = append(event.Projects, project.Id)
	return db.Update(func(tx *bolt.Tx) error {
		eventBucket := tx.Bucket([]byte(EVENTS_BUCKET))

		eventBytes, err := json.Marshal(event)
		if err != nil {
			return err
		}
		return eventBucket.Put(event.Id.Bytes(), eventBytes)
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
		return commitBucket.Put(eventId.Bytes(), emptyBytes)
	})
}
