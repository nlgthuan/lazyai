package skydeck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
)

type SkyDeckClient struct{}

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

func getConversationID(conversationID int, resp *SendMessageResponse) int {
	if conversationID != 0 {
		return conversationID
	}
	return resp.Data.ConversationID
}

type StreamingReq struct {
	MessageID int `json:"message_id"`
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
