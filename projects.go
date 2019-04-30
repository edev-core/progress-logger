package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	uuid "github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
	"gopkg.in/src-d/go-git.v4"
	//"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type Project struct {
	Id             uuid.UUID `json:"id"`
	URL            string    `json:"url" binding:"required"`
	Name           string    `json:"name" binding:"required"`
	Authors        []string  `json:"authors" binding:"required"`
	Path           string    `json:"path"`
	LastCommit     string    `json:"last_commit"`
	LastCommitTime time.Time `json:"last_commit_date"`
}

func (p *Project) RetrieveNewCommits(db *bolt.DB, eventId *uuid.UUID) error {
	repo, err := git.PlainOpen(p.Path)
	if err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}
	pullOpts := new(git.PullOptions)
	err = worktree.Pull(pullOpts)
	if err != git.NoErrAlreadyUpToDate {
		return err
	}
	cIter, err := repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime})
	if err != nil {
		return err
	}

	lastCommitTime := p.LastCommitTime
	err = cIter.ForEach(func(c *object.Commit) error {
		if c.Author.When.After(p.LastCommitTime) {
			p.LastCommit = c.Hash.String()
			p.LastCommitTime = c.Author.When
			fmt.Println("Most recent ?")
		}
		if !c.Author.When.After(lastCommitTime) {
			return nil
		}
		fmt.Println(c)
		dbStoreNewCommit(db,
			&Commit{Project: p.Name,
				Author:  c.Author.Name,
				Message: c.Message,
				Date:    c.Author.When},
			eventId)

		return nil
	})
	if err != nil {
		return err
	}
	fmt.Println("Retained Last commit: ", p.LastCommit)
	return dbStoreProject(db, p)
}

func RegisterProject(db *bolt.DB, eventId *uuid.UUID, project *Project) (*uuid.UUID, error) {
	// Checking that the url is valid
	projectUrl, err := url.Parse(project.URL)
	if err != nil {
		return nil, err
	}
	// Checks if a project with the same url does not exist
	_, err = dbGetProject(db, &project.Id)
	if _, ok := err.(*EventNotFound); !ok {
		return nil, err
	}

	// Creating the file path where the git repo will be stored
	project.Path = filepath.Join(GIT_PATH, eventId.String(), projectUrl.Hostname(), projectUrl.Path)
	err = os.MkdirAll(project.Path, os.ModePerm)
	if err != nil {
		return nil, err
	}
	projectId, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	project.Id = projectId

	// Cloning the git repo
	repo, err := git.PlainClone(project.Path, false, &git.CloneOptions{
		URL: project.URL,
	})
	if err != nil {
		return nil, err
	}
	// Fetching all commits since the first one
	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}
	project.LastCommit = ref.Hash().String()
	err = project.RetrieveNewCommits(db, eventId)
	if err != nil {
		return nil, err
	}

	err = dbStoreProject(db, project)
	dbAddProjectToEvent(db, eventId, project)

	return &projectId, nil
}
