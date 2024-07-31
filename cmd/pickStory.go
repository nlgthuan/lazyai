package cmd

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var owner string = "TN"
var selectedState string = "started"

// Story represents a Pivotal Tracker story
type Story struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Desc string `json:"description"`
}

func (s Story) FilterValue() string {
	return s.Name
}

// pickStoryCmd represents the pickStory command
var pickStoryCmd = &cobra.Command{
	Use:   "pickStory",
	Short: "Filter Pivotal Tracker story you are working on and return its ID",
	Run: func(cmd *cobra.Command, args []string) {
		// Make HTTP request to Pivotal Tracker API
		url := "https://www.pivotaltracker.com/services/v5/projects/" + projectID + "/stories?filter=owner%3A%22TN%22"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("X-TrackerToken", apiToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Fatalf("Failed to get stories: %s", string(bodyBytes))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read response body: %v", err)

		}

		var stories []Story
		err = json.Unmarshal(bodyBytes, &stories)
		if err != nil {
			log.Fatalf("Failed to unmarshal response: %v", err)
		}

		myOptions := make([]huh.Option[string], len(stories))

		for i, story := range stories {
			myOptions[i] = huh.NewOption(story.Name, story.Desc)
		}

		var toppings string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Pick a story.").
					Options(myOptions...).
					Value(&toppings),
			),
		)
		form.Run()
		fmt.Print(toppings)

	},
}

func init() {
	rootCmd.AddCommand(pickStoryCmd)
}
