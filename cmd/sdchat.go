package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	BaseURL     = "https://admin.skydeck.ai"
	ReferrerURL = "https://eastagile.skydeck.ai/"
)

type Config struct {
	AccessToken  string
	RefreshToken string
}

type SendMessagePayload struct {
	Message             string `json:"message"`
	ModelID             int    `json:"model_id"`
	ConversationID      *int   `json:"conversation_id,omitempty"`
	RegenerateMessageID int    `json:"regenerate_message_id"`
	NonAI               bool   `json:"non_ai"`
}

type SendMessageResponse struct {
	Data struct {
		ConversationID       int                    `json:"conversation_id"`
		AssistantMessageID   int                    `json:"assistant_message_id"`
		RememberizerAPIQuery map[string]interface{} `json:"rememberizer_api_query"`
	} `json:"data"`
}

func init() {
	rootCmd.AddCommand(sdchatCmd)
	sdchatCmd.Flags().IntP("conversation", "c", 0, "Conversation ID to use for the message")
	sdchatCmd.Flags().BoolP("open", "o", false, "Open the conversation in the default browser instead of streaming the response to the terminal")
}

var sdchatCmd = &cobra.Command{
	Use:   "sdchat",
	Short: "Send a message and get a streaming response from the server",
	Long: `Configuration:
The command requires an access token and a refresh token to authenticate with the SkyDeck API.
These tokens should be specified in the ~/.lazyai.yml configuration file under the 'skydeck' section:

skydeck:
    accessToken: <your access token>
    refreshToken: <your refresh token>

Examples:
    # Send a message to the SkyDeck AI service
    sdchat "Hello, SkyDeck!"

    # Send a message to a specific conversation
    sdchat -c 123 "Continue our previous conversation."

    # Send a message and open the conversation in the default browser
    sdchat -o "Hello, SkyDeck!"
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadConfig()
		if err != nil {
			return err
		}

		cmd.Flags().String("accessToken", config.AccessToken, "SkyDeck access token")
		cmd.Flags().String("refreshToken", config.RefreshToken, "SkyDeck refresh token")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		accessToken, _ := cmd.Flags().GetString("accessToken")
		refreshToken, _ := cmd.Flags().GetString("refreshToken")
		conversationID, _ := cmd.Flags().GetInt("conversation")
		openInBrowser, _ := cmd.Flags().GetBool("open")

		if len(args) < 1 {
			fmt.Println("Please provide a message to send")
			return
		}
		message := args[0]

		var conversationIDPtr *int
		if conversationID != 0 {
			conversationIDPtr = &conversationID
		}

		payload := SendMessagePayload{
			Message:             message,
			ModelID:             4094,
			ConversationID:      conversationIDPtr,
			RegenerateMessageID: -1,
			NonAI:               false,
		}

		apiClient := NewAPIClient(accessToken, refreshToken)

		resp, err := apiClient.sendMessage(payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
			return
		}

		convoID := 0
		if conversationID != 0 {
			convoID = conversationID
		} else {
			convoID = resp.Data.ConversationID
		}

		conversationURL := fmt.Sprintf("https://eastagile.skydeck.ai/conversations/%d", convoID)

		if openInBrowser {
			if err := openURL(conversationURL); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening URL: %v\n", err)
			}
		} else {
			err = apiClient.getStreamingResponse(resp.Data.AssistantMessageID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting streaming response: %v\n", err)
				return
			}

			fmt.Printf("\nVisit the conversation at: %s", conversationURL)
		}
	},
}

// func main() {
// 	if err := rootCmd.Execute(); err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// }

func loadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error finding home directory: %w", err)
	}

	viper.SetConfigName(".lazyai")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := &Config{
		AccessToken:  viper.GetString("skydeck.accessToken"),
		RefreshToken: viper.GetString("skydeck.refreshToken"),
	}

	if config.AccessToken == "" || config.RefreshToken == "" {
		return nil, fmt.Errorf("accessToken and refreshToken must be set in the configuration file.\nPlease check your ~/.lazyai.yml again!\n")
	}

	return config, nil
}

func updateAccessToken(newAccessToken string) error {
	viper.Set("skydeck.accessToken", newAccessToken)
	return viper.WriteConfig()
}

type APIClient struct {
	Client       *http.Client
	AccessToken  string
	RefreshToken string
}

func NewAPIClient(accessToken, refreshToken string) *APIClient {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	return &APIClient{
		Client:       client,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func (api *APIClient) sendMessage(payload SendMessagePayload) (*SendMessageResponse, error) {
	url := BaseURL + "/api/v1/conversations/chat_v2/"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("message", payload.Message)
	writer.WriteField("model_id", fmt.Sprintf("%d", payload.ModelID))
	if payload.ConversationID != nil {
		writer.WriteField("conversation_id", fmt.Sprintf("%d", *payload.ConversationID))
	}
	writer.WriteField("regenerate_message_id", fmt.Sprintf("%d", payload.RegenerateMessageID))
	writer.WriteField("non_ai", fmt.Sprintf("%t", payload.NonAI))

	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Referer", ReferrerURL)
	req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: api.AccessToken})
	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: api.RefreshToken})

	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		api.AccessToken, err = api.refreshTokens()
		if err != nil {
			return nil, fmt.Errorf("error refreshing tokens: %v", err)
		}

		req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: api.AccessToken})
		resp, err = api.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error decoding response: %v, body: %s", err, string(bodyBytes))
	}

	return &response, nil
}

func (api *APIClient) getStreamingResponse(messageID int) error {
	url := BaseURL + fmt.Sprintf("/api/v1/conversations/streaming/?message_id=%d", messageID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Referer", ReferrerURL)
	req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: api.AccessToken})
	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: api.RefreshToken})

	resp, err := api.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		api.AccessToken, err = api.refreshTokens()
		if err != nil {
			return fmt.Errorf("error refreshing tokens: %v", err)
		}

		req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: api.AccessToken})
		resp, err = api.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func (api *APIClient) refreshTokens() (string, error) {
	url := BaseURL + "/api/v1/authentication/token/refresh/"

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: api.RefreshToken})
	req.Header.Set("Referer", ReferrerURL)

	resp, err := api.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var newAccess string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "eastagile_access" {
			newAccess = cookie.Value
		}
	}

	if err := updateAccessToken(newAccess); err != nil {
		return "", fmt.Errorf("error writing config file: %v", err)
	}

	return newAccess, nil
}

func openURL(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}
