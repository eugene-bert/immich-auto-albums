package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eugene-bert/immich-auto-albums/rules"
)

type Client struct {
	URL    string
	APIKey string
}

type searchResponse struct {
	Assets struct {
		Items    []asset `json:"items"`
		NextPage *int    `json:"nextPage"`
	} `json:"assets"`
}

type asset struct {
	ID string `json:"id"`
}

type albumResponse struct {
	ID      string  `json:"id"`
	Name    string  `json:"albumName"`
	Assets  []asset `json:"assets"`
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.URL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
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
		page = *resp.Assets.NextPage
	}
	return allIDs, nil
}

func (c *Client) FindAlbum(name string) (string, error) {
	data, err := c.do("GET", "/api/albums", nil)
	if err != nil {
		return "", err
	}
	var albums []albumResponse
	json.Unmarshal(data, &albums)
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
	json.Unmarshal(data, &a)
	return a.ID, nil
}

func (c *Client) GetAlbumAssetIDs(albumID string) (map[string]bool, error) {
	data, err := c.do("GET", "/api/albums/"+albumID, nil)
	if err != nil {
		return nil, err
	}
	var a albumResponse
	json.Unmarshal(data, &a)
	ids := make(map[string]bool, len(a.Assets))
	for _, asset := range a.Assets {
		ids[asset.ID] = true
	}
	return ids, nil
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
