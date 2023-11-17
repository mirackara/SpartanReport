package spartanreport

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EventsHome struct {
	PreviousSeason Season `json:"PreviousSeason"`
	CurrentSeason  Season `json:"CurrentSeason"`
}

func HandleEventsHome(c *gin.Context) {
	fmt.Println("HandleEventsHome")
	if !cachedSeasons.IsEmpty() {
		var activeIndex int
		for i, season := range cachedSeasons.Seasons.Seasons {
			if season.IsActive {
				activeIndex = i
				break
			}
		}

		// Check if the active season is not the first one in the list
		if activeIndex > 0 {
			EventsToReturn := EventsHome{
				PreviousSeason: cachedSeasons.Seasons.Seasons[activeIndex],
				CurrentSeason:  cachedSeasons.Seasons.Seasons[activeIndex+1],
			}
			c.JSON(http.StatusOK, EventsToReturn)
			return
		}

		// Handle case where the active season is the first in the list
		EventsToReturn := EventsHome{
			PreviousSeason: cachedSeasons.Seasons.Seasons[activeIndex], // or set a default value
			CurrentSeason:  cachedSeasons.Seasons.Seasons[activeIndex+1],
		}
		c.JSON(http.StatusOK, EventsToReturn)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while getting events"})
}
