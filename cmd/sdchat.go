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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Define constants for referrer URL, access token, and refresh token
const (
	BaseURL     = "https://admin.skydeck.ai"
	ReferrerURL = "https://eastagile.skydeck.ai/"
)

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

		accessToken := viper.GetString("skydeck.accessToken")
		refreshToken := viper.GetString("skydeck.refreshToken")

		if accessToken == "" || refreshToken == "" {
			return fmt.Errorf("accessToken and refreshToken must be set in the configuration file.\nPlease check your ~/.lazyai.yml again!\n")
		}

		cmd.Flags().String("accessToken", accessToken, "SkyDeck access token")
		cmd.Flags().String("refreshToken", refreshToken, "SkyDeck refresh token")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		accessToken, _ := cmd.Flags().GetString("accessToken")
		refreshToken, _ := cmd.Flags().GetString("refreshToken")
		conversationID, _ := cmd.Flags().GetInt("conversation")

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

		// Send message
		resp, err := sendMessage(payload, accessToken, refreshToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
			return
		}

		// Get streaming response
		err = getStreamingResponse(resp.Data.AssistantMessageID, accessToken, refreshToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting streaming response: %v\n", err)
			return
		}

		convoID := 0
		if conversationID != 0 {
			convoID = conversationID
		} else {
			convoID = resp.Data.ConversationID
		}

		fmt.Printf("\nVisit the conversation at: https://eastagile.skydeck.ai/conversations/%d", convoID)
	},
}

func sendMessage(payload SendMessagePayload, access, refresh string) (*SendMessageResponse, error) {
	url := BaseURL + "/api/v1/conversations/chat_v2/"

	// Create a buffer to write our form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	writer.WriteField("message", payload.Message)
	writer.WriteField("model_id", fmt.Sprintf("%d", payload.ModelID))
	if payload.ConversationID != nil {
		writer.WriteField("conversation_id", fmt.Sprintf("%d", *payload.ConversationID))
	}
	writer.WriteField("regenerate_message_id", fmt.Sprintf("%d", payload.RegenerateMessageID))
	writer.WriteField("non_ai", fmt.Sprintf("%t", payload.NonAI))

	// Close the writer to finalize the form data
	writer.Close()

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Referer", ReferrerURL)

	// Set cookies
	cookieJar, _ := cookiejar.New(nil)
	client.Jar = cookieJar
	req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: access})
	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: refresh})

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// If the status code is 401 Unauthorized, refresh the tokens and retry
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("Refreshing token....")
		access, err = refreshTokens(refresh)
		if err != nil {
			return nil, fmt.Errorf("error refreshing tokens: %v", err)
		}

		// Retry the request with new tokens
		req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: access})
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	// Check if the status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var sendMessageResponse SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&sendMessageResponse); err != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error decoding response: %v, body: %s", err, string(bodyBytes))
	}

	return &sendMessageResponse, nil
}

func getStreamingResponse(messageID int, access, refresh string) error {
	url := BaseURL + fmt.Sprintf("/api/v1/conversations/streaming/?message_id=%d", messageID)
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Referer", ReferrerURL)

	// Set cookies
	cookieJar, _ := cookiejar.New(nil)
	client.Jar = cookieJar
	req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: access})
	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: refresh})

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If the status code is 401 Unauthorized, refresh the tokens and retry
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Println("Refreshing token....")
		access, err = refreshTokens(refresh)
		if err != nil {
			return fmt.Errorf("error refreshing tokens: %v", err)
		}

		// Retry the request with new tokens
		req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: access})
		resp, err = client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	// Check if the status code is not 200 OK
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Stream the response
	_, err = io.Copy(os.Stdout, resp.Body)

	return err
}

func refreshTokens(currentRefreshToken string) (string, error) {
	url := BaseURL + "/api/v1/authentication/token/refresh/"
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}

	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: currentRefreshToken})
	req.Header.Set("Referer", ReferrerURL)

	resp, err := client.Do(req)
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

	// Update the tokens in the configuration
	viper.Set("skydeck.accessToken", newAccess)
	if err := viper.WriteConfig(); err != nil {
		return "", fmt.Errorf("error writing config file: %v", err)
	}

	return newAccess, nil
}
