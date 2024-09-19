package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var selectedState string = "started"

// Story represents a Pivotal Tracker story
type Story struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Desc string `json:"description"`
	URL  string `json:"url"`
}

func (s Story) FilterValue() string {
	return s.Name
}

// pickPTCmd represents the pickPT command
var pickPTCmd = &cobra.Command{
	Use:   "pickPT",
	Short: "Retrieve the description of your active Pivotal Tracker story",
	Long: `The pickPT command helps you quickly find the Pivotal Tracker story you are currently working on (in 'started' state) and returns its description.
This command streamlines your workflow by providing instant access to essential story details.

Configuration:
Ensure your configuration file (~/.lazyai.yml) is set up properly with the following details:

    pivotalTracker:
        apiToken: <your_api_token>
        projectID: <project_ID>
        owner: <your_account_name, e.g. thuanngo>
`,

	PreRunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error finding home directory: %w", err)
		}

		viper.SetConfigName(".lazyai")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(home)

		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}

		apiToken := viper.GetString("pivotalTracker.apiToken")
		projectID := viper.GetString("pivotalTracker.projectID")
		owner := viper.GetString("pivotalTracker.owner")

		if apiToken == "" || projectID == "" || owner == "" {
			return fmt.Errorf("apiToken, projectID and owner must be set in the configuration file.\nPlease check your ~/.lazyai.yml again!\n")
		}

		cmd.Flags().String("apiToken", apiToken, "Pivotal Tracker API token")
		cmd.Flags().String("projectID", projectID, "Pivotal Tracker project ID")
		cmd.Flags().String("owner", owner, "Pivotal Tracker owner user")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		apiToken, _ := cmd.Flags().GetString("apiToken")
		projectID, _ := cmd.Flags().GetString("projectID")
		owner, _ := cmd.Flags().GetString("owner")

		link, _ := cmd.Flags().GetBool("link")

		// Make HTTP request to Pivotal Tracker API
		baseURL := "https://www.pivotaltracker.com/services/v5/projects/" + projectID + "/stories"
		queryParams := url.Values{}
		queryParams.Add("filter", fmt.Sprintf("owner:\"%s\" AND state:\"%s\"", owner, selectedState))
		encodedURL := fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())

		req, err := http.NewRequest("GET", encodedURL, nil)
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
			if link != true {
				myOptions[i] = huh.NewOption(story.Name, story.Desc)
			} else {
				myOptions[i] = huh.NewOption(story.Name, story.URL)
			}
		}

		var desc string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Pick a story.").
					Options(myOptions...).
					Value(&desc),
			),
		)

		form.Run()
		fmt.Print(desc)
	},
}

func init() {
	rootCmd.AddCommand(pickPTCmd)
	pickPTCmd.Flags().BoolP("link", "l", false, "Returns only the link of the story")
}
