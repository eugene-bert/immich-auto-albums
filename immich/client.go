package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eugene-bert/immich-auto-albums/rules"
)

type Client struct {
	URL    string
	APIKey string
	HTTP   *http.Client
}

var defaultHTTP = &http.Client{Timeout: 30 * time.Second}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return defaultHTTP
}

type searchResponse struct {
	Assets struct {
		Items    []asset          `json:"items"`
		NextPage *json.RawMessage `json:"nextPage"`
	} `json:"assets"`
}

type asset struct {
	ID string `json:"id"`
}

type albumResponse struct {
	ID     string  `json:"id"`
	Name   string  `json:"albumName"`
	Assets []asset `json:"assets"`
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.URL, "/")+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) SearchMetadata(rule rules.Rule) ([]string, error) {
	filters := make(map[string]any)
	if rule.CameraMake != "" {
		filters["make"] = rule.CameraMake
	}
	if rule.CameraModel != "" {
		filters["model"] = rule.CameraModel
	}
	if rule.LensModel != "" {
		filters["lensModel"] = rule.LensModel
	}
	if rule.MediaType != "" {
		filters["type"] = rule.MediaType
	}
	if rule.City != "" {
		filters["city"] = rule.City
	}
	if rule.State != "" {
		filters["state"] = rule.State
	}
	if rule.Country != "" {
		filters["country"] = rule.Country
	}
	if rule.TakenAfter != "" {
		filters["takenAfter"] = rule.TakenAfter
	}
	if rule.TakenBefore != "" {
		filters["takenBefore"] = rule.TakenBefore
	}
	if rule.OriginalFileName != "" {
		filters["originalFileName"] = rule.OriginalFileName
	}
	if rule.Description != "" {
		filters["description"] = rule.Description
	}

	var allIDs []string
	page := 1
	for {
		filters["page"] = page
		filters["size"] = 1000

		data, err := c.do("POST", "/api/search/metadata", filters)
		if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}

		var resp searchResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse search: %w", err)
		}

		for _, a := range resp.Assets.Items {
			allIDs = append(allIDs, a.ID)
		}

		if resp.Assets.NextPage == nil {
			break
		}
		raw := string(*resp.Assets.NextPage)
		raw = strings.Trim(raw, "\"")
		next, err := strconv.Atoi(raw)
		if err != nil {
			break
		}
		page = next
	}
	return allIDs, nil
}

func (c *Client) FindAlbum(name string) (string, error) {
	data, err := c.do("GET", "/api/albums", nil)
	if err != nil {
		return "", err
	}
	var albums []albumResponse
	if err := json.Unmarshal(data, &albums); err != nil {
		return "", fmt.Errorf("parse albums: %w", err)
	}
	for _, a := range albums {
		if a.Name == name {
			return a.ID, nil
		}
	}
	return "", nil
}

func (c *Client) CreateAlbum(name string) (string, error) {
	data, err := c.do("POST", "/api/albums", map[string]string{"albumName": name})
	if err != nil {
		return "", err
	}
	var a albumResponse
	if err := json.Unmarshal(data, &a); err != nil {
		return "", fmt.Errorf("parse album: %w", err)
	}
	return a.ID, nil
}

func (c *Client) GetAlbumAssetIDs(albumID string) (map[string]bool, error) {
	data, err := c.do("GET", "/api/albums/"+albumID, nil)
	if err != nil {
		return nil, err
	}
	var a albumResponse
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("parse album: %w", err)
	}
	ids := make(map[string]bool, len(a.Assets))
	for _, asset := range a.Assets {
		ids[asset.ID] = true
	}
	return ids, nil
}

func (c *Client) Ping() error {
	_, err := c.do("GET", "/api/users/me", nil)
	return err
}

type ExploreField struct {
	FieldName string       `json:"fieldName"`
	Values    []FieldValue `json:"values"`
}

type FieldValue struct {
	Value string `json:"value"`
	Count int    `json:"data"`
}

type ExploreData struct {
	CameraMakes  []string
	CameraModels []string
	Cities       []string
	States       []string
	Countries    []string
	AlbumNames   []string
}

func (c *Client) Explore() (ExploreData, error) {
	data, err := c.do("GET", "/api/search/explore", nil)
	if err != nil {
		return ExploreData{}, err
	}
	var fields []ExploreField
	if err := json.Unmarshal(data, &fields); err != nil {
		return ExploreData{}, fmt.Errorf("parse explore: %w", err)
	}

	var result ExploreData
	for _, f := range fields {
		vals := make([]string, 0, len(f.Values))
		for _, v := range f.Values {
			if v.Value != "" {
				vals = append(vals, v.Value)
			}
		}
		switch f.FieldName {
		case "exifInfo.make":
			result.CameraMakes = vals
		case "exifInfo.model":
			result.CameraModels = vals
		case "exifInfo.city":
			result.Cities = vals
		case "exifInfo.state":
			result.States = vals
		case "exifInfo.country":
			result.Countries = vals
		}
	}
	return result, nil
}

func (c *Client) ListAlbumNames() ([]string, error) {
	data, err := c.do("GET", "/api/albums", nil)
	if err != nil {
		return nil, err
	}
	var albums []albumResponse
	if err := json.Unmarshal(data, &albums); err != nil {
		return nil, fmt.Errorf("parse albums: %w", err)
	}
	names := make([]string, 0, len(albums))
	for _, a := range albums {
		names = append(names, a.Name)
	}
	return names, nil
}

func (c *Client) AddAssetsToAlbum(albumID string, assetIDs []string) error {
	for i := 0; i < len(assetIDs); i += 500 {
		end := i + 500
		if end > len(assetIDs) {
			end = len(assetIDs)
		}
		batch := assetIDs[i:end]
		_, err := c.do("PUT", "/api/albums/"+albumID+"/assets", map[string]any{"ids": batch})
		if err != nil {
			return err
		}
	}
	return nil
}
