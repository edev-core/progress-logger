package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	// "encoding/json"
	"github.com/satori/go.uuid"
	"strconv"
)

const (
	GIT_PATH = "repos"
)

type TrackingCommand int

const (
	STOP_TRACKING TrackingCommand = 1
	POLL          TrackingCommand = 2
)

func main() {
	r := gin.Default()
	main_key := "42"
	trackedEvents := make(map[uuid.UUID]chan TrackingCommand)
	quit := make(chan struct{})

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

		eventId, err := CreateEvent(db, &eventRequest, main_key)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		c.JSON(200, gin.H{
			"eventId": *eventId,
		})
	})

	r.POST("/api/event/:eventId/projects", func(c *gin.Context) {
		// Check that the eventId is a valid UUID
		eventId, err := uuid.FromString(c.Param("eventId"))
		if err != nil {
			c.AbortWithStatus(404)
			return
		}

		// Check if the provided project is valid JSON
		var project Project
		if err := c.ShouldBindJSON(&project); err != nil {
			c.AbortWithStatus(400)
			return
		}

		// Registers a new project
		projectId, err := RegisterProject(db, &eventId, &project)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		c.JSON(200, gin.H{
			"projectId": *projectId,
		})
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
		commits, err := FetchCommits(db, page, limit, &eventId)
		if err != nil {
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

	r.PUT("/api/event/:eventId", func(c *gin.Context) {
		eventId, err := uuid.FromString(c.Param("eventId"))
		if err != nil {
			c.AbortWithStatus(400)
			return
		}
		// Check if the eventModification is valid JSON
		var eventMod EventModification
		if err := c.ShouldBindJSON(&eventMod); err != nil {
			c.AbortWithStatus(400)
			return
		}

		if eventMod.Track {
			trackedEvents[eventId] = make(chan TrackingCommand)
			go TrackEvent(db, &eventId, trackedEvents[eventId], quit)
		} else {
			trackedEvents[eventId] <- STOP_TRACKING
			delete(trackedEvents, eventId)
		}

		c.JSON(200, gin.H{})
	})
	r.Run()
}
