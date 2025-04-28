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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
		ConversationID int `json:"conversation_id"`
		Messages       []struct {
			ID        int    `json:"id"`
			Type      string `json:"type"`
			Content   string `json:"content"`
			Streaming bool   `json:"streaming"`
		} `json:"messages"`
	} `json:"data"`
}

type Config struct {
	accessToken    string
	currentConvoID int
	refreshToken   string
}

var (
	config         *Config
	conversationID int
	openInBrowser  bool
	newConvo       bool
)

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
	Run: func(cmd *cobra.Command, args []string) {
		handleRun(cmd, args)
	},
}

func init() {
	cobra.OnInitialize(loadConfig)

	sdchatCmd.Flags().IntVarP(&conversationID, "conversation", "c", 0, "Conversation ID to use for the message")
	sdchatCmd.Flags().BoolVarP(&openInBrowser, "open", "o", false, "Open the conversation in the default browser instead of streaming the response to the terminal")
	sdchatCmd.Flags().BoolVarP(&newConvo, "new", "n", false, "Chat in a new conversation")

	rootCmd.AddCommand(sdchatCmd)
}

func loadConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	viper.SetConfigName(".lazyai")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(home)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	cobra.CheckErr(err)

	config = &Config{
		accessToken:    viper.GetString("skydeck.accessToken"),
		refreshToken:   viper.GetString("skydeck.refreshToken"),
		currentConvoID: viper.GetInt("skydeck.convoID"),
	}
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

	writeFormFields(writer, payload)
	writer.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, err
	}

	setRequestHeaders(req, api.AccessToken, api.RefreshToken, writer.FormDataContentType())
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return handleResponse(resp, api, payload)
}

func writeFormFields(writer *multipart.Writer, payload SendMessagePayload) {
	writer.WriteField("message", payload.Message)
	writer.WriteField("model_id", fmt.Sprintf("%d", payload.ModelID))
	if payload.ConversationID != nil {
		writer.WriteField("conversation_id", fmt.Sprintf("%d", *payload.ConversationID))
	}
	writer.WriteField("regenerate_message_id", fmt.Sprintf("%d", payload.RegenerateMessageID))
	writer.WriteField("non_ai", fmt.Sprintf("%t", payload.NonAI))
}

func setRequestHeaders(req *http.Request, accessToken, refreshToken, contentType string) {
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Referer", ReferrerURL)
	req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: accessToken})
	req.AddCookie(&http.Cookie{Name: "eastagile_refresh", Value: refreshToken})
}

func handleResponse(resp *http.Response, api *APIClient, payload SendMessagePayload) (*SendMessageResponse, error) {
	var response SendMessageResponse
	if resp.StatusCode == http.StatusUnauthorized {
		return handleUnauthorizedResponse(api, payload)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, readResponseBody(resp))
	}

	err := json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &response, nil
}

func handleUnauthorizedResponse(api *APIClient, payload SendMessagePayload) (*SendMessageResponse, error) {
	newAccessToken, err := api.refreshTokens()
	if err != nil {
		return nil, fmt.Errorf("error refreshing tokens: %v", err)
	}
	api.AccessToken = newAccessToken

	return api.sendMessage(payload)
}

func readResponseBody(resp *http.Response) string {
	bodyBytes, _ := io.ReadAll(resp.Body)
	return string(bodyBytes)
}

func handleRun(cmd *cobra.Command, args []string) {
	var message string
	var err error

	// Check if there is data coming from stdin
	if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		// Read from standard input
		inputBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			return
		}
		message = string(inputBytes)
	} else if len(args) > 0 {
		// Use the first argument as the message
		message = args[0]
	} else {
		fmt.Println("Please provide a message to send either as an argument or through stdin")
		return
	}

	// Trim message to remove any trailing newlines or spaces
	message = strings.TrimSpace(message)

	// Handle conversation
	var conversationIDPtr *int
	if config.currentConvoID != 0 {
		conversationIDPtr = &config.currentConvoID
	}
	if conversationID != 0 {
		conversationIDPtr = &conversationID
	}
	if newConvo {
		conversationIDPtr = nil
	}

	payload := SendMessagePayload{
		Message:             message,
		ModelID:             4094,
		ConversationID:      conversationIDPtr,
		RegenerateMessageID: -1,
		NonAI:               false,
	}

	apiClient := NewAPIClient(config.accessToken, config.refreshToken)
	resp, err := apiClient.sendMessage(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
		return
	}

	convoID := getConversationID(conversationID, resp)
	viper.Set("skydeck.convoID", convoID)
	viper.WriteConfig()
	conversationURL := fmt.Sprintf("https://eastagile.skydeck.ai/conversations/%d", convoID)

	if openInBrowser {
		if err := openURL(conversationURL); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening URL: %v\n", err)
		}
	} else {
		// Find the assistant message ID in the response
		var assistantMessageID int
		for _, msg := range resp.Data.Messages {
			if msg.Type == "assistant" && msg.Streaming {
				assistantMessageID = msg.ID
				break
			}
		}
		
		if assistantMessageID == 0 {
			fmt.Fprintf(os.Stderr, "Error: No streaming assistant message found in the response\n")
			return
		}
		
		err = apiClient.getStreamingResponse(assistantMessageID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting streaming response: %v\n", err)
			return
		}
	}
}

func getConversationID(conversationID int, resp *SendMessageResponse) int {
	if conversationID != 0 {
		return conversationID
	}
	return resp.Data.ConversationID
}

type StreamingReq struct {
	MessageID int `json:"message_id"`
}

type StreamingResponse struct {
	Data struct {
		ConversationID int `json:"conversation_id"`
		Messages       []struct {
			ID        int    `json:"id"`
			Type      string `json:"type"`
			Content   string `json:"content"`
			Streaming bool   `json:"streaming"`
		} `json:"messages"`
	} `json:"data"`
}

func (api *APIClient) getStreamingResponse(messageID int) error {
	url := BaseURL + fmt.Sprintf("/api/v1/conversations/streaming/")

	payload := StreamingReq{
		MessageID: messageID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	setRequestHeaders(req, api.AccessToken, api.RefreshToken, "application/json")

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

		updateAccessToken(api.AccessToken)

		req.AddCookie(&http.Cookie{Name: "eastagile_access", Value: api.AccessToken})
		resp, err = api.Client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, readResponseBody(resp))
	}

	// The response might be the raw streaming content directly
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	
	bodyStr := string(bodyBytes)
	
	// Try to decode as JSON first
	var streamResp StreamingResponse
	if err := json.Unmarshal(bodyBytes, &streamResp); err == nil {
		// If successful JSON decode, find the assistant message
		for _, msg := range streamResp.Data.Messages {
			if msg.Type == "assistant" && msg.Streaming {
				fmt.Print(msg.Content)
				return nil
			}
		}
		return fmt.Errorf("no streaming assistant message found in the response")
	}
	
	// If not valid JSON, output the raw response as it's likely the streamed content
	fmt.Print(bodyStr)
	return nil
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
		return "", fmt.Errorf("received non-200 response code: %d, body: %s", resp.StatusCode, readResponseBody(resp))
	}

	return extractAccessTokenFromCookies(resp.Cookies()), nil
}

func extractAccessTokenFromCookies(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == "eastagile_access" {
			return cookie.Value
		}
	}
	return ""
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
