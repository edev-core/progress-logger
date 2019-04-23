package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	// "github.com/src-d/go-git"
	// "encoding/json"
	"github.com/satori/go.uuid"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	GIT_PATH = "repos"
)

type Project struct {
	URL     string   `json:"url" binding:"required"`
	Name    string   `json:"name" binding:"required"`
	Authors []string `json:"authors" binding:"required"`
	Path    string   `json:"path"`
}

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

func main() {
	r := gin.Default()
	main_key := "42"

	db, err := dbOpen("db/progress-logger.db")
	if err != nil {
		fmt.Println("Error opening DB: ", err)
		return
	}

	r.POST("/api/event", func(c *gin.Context) {
		var eventRequest EventRequest
		if err := c.ShouldBindJSON(&eventRequest); err != nil {
			c.AbortWithError(500, err)
			return
		}

		if eventRequest.Name == "" {
			c.AbortWithStatus(400)
			return
		}
		if eventRequest.Key != main_key {
			c.AbortWithStatus(403)
			return
		}

		eventId, err := uuid.NewV4()
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		eventPath := filepath.Join(GIT_PATH, eventId.ToString())
		err = os.Mkdir(eventPath, os.ModePerm)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		err = initEvent(db, &eventId, &eventRequest.Name)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, gin.H{
			"eventId": eventId,
		})
	})

	r.POST("/api/event/:eventId/project", func(c *gin.Context) {
		eventId, err := uuid.FromString(c.Param("eventId"))
		if err != nil {
			c.AbortWithStatus(404)
			return
		}
		var project Project
		if err := c.ShouldBindJSON(&project); err != nil {
			c.AbortWithError(500, err)
			return
		}
		projectUrl, err := url.Parse(project.URL)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		project.Path = filepath.Join(GIT_PATH, eventId.ToString(), projectUrl.Hostname, projectUrl.Path)
		err = os.MkdirAll(project.Path, os.ModePerm)
		if err != nil {
			c.AbortWithError(500, err)
			return err
		}

		_, err = dbGetProject(db, &project.URL)
		if _, ok := err.(*EventNotFound); ok {
			err := dbStoreProject(db, &project)
			if err != nil {
				return
			}
			dbAddProjectToEvent(db, &eventId, &project)
		}

		c.JSON(200, gin.H{})
	})

	r.GET("/api/event/:eventId/commits", func(c *gin.Context) {
		rawPage := c.DefaultQuery("page", "0")
		rawLimit := c.DefaultQuery("limit", "25")

		pageInt, _ := strconv.Atoi(rawPage)
		limitInt, _ := strconv.Atoi(rawLimit)
		page := uint32(pageInt)
		limit := uint32(limitInt)

		eventId, err := uuid.FromString(c.Param("eventId"))
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
		commitCount := uint32(len(commits))
		start := page * limit
		if start > commitCount {
			commits = nil
		} else {
			if start+limit > commitCount {
				commits = commits[start:]
			} else {
				commits = commits[start : start+limit]
			}
		}

		c.JSON(200, gin.H{
			"commits": commits,
		})
	})

	r.GET("/api/event/:eventId", func(c *gin.Context) {
		eventId, err := uuid.FromString(c.Param("eventId"))
		if err != nil {
			c.AbortWithStatus(400)
			return
		}
		event, err := dbGetEvent(db, &eventId)
		if _, ok := err.(*EventNotFound); ok {
			c.AbortWithStatus(404)
		} else if err != nil {
			c.AbortWithError(500, err)
			return
		}

		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		c.JSON(200, event)
	})
	r.Run()
}
