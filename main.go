package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	// "github.com/src-d/go-git"
	// "encoding/json"
	"github.com/satori/go.uuid"
	"time"
)

type Project struct {
	URL     string   `json:"url" binding:"required"`
	Name    string   `json:"name" binding:"required"`
	Authors []string `json:"authors" binding:"required"`
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
		err = initEvent(db, &eventId, &eventRequest.Name)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		c.JSON(200, gin.H{
			"eventId": eventId,
		})
	})

	r.PUT("/api/event/:eventId/register", func(c *gin.Context) {
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
