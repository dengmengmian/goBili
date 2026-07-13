package parser

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dengmengmian/goBili/auth"
	"github.com/sirupsen/logrus"
)

func TestExtractBVID(t *testing.T) {
	tests := []struct {
		url      string
		wantBVID string
	}{
		{"https://www.bilibili.com/video/BV1qt4y1X7TW", "BV1qt4y1X7TW"},
		{"https://www.bilibili.com/video/BV1xx411c7mD/?spm_id_from=333.337.0.0", "BV1xx411c7mD"},
		{"https://b23.tv/BV1qt4y1X7TW", "BV1qt4y1X7TW"},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			re := regexp.MustCompile(`BV[a-zA-Z0-9]+`)
			got := re.FindString(tt.url)
			if got != tt.wantBVID {
				t.Errorf("extractBVID(%q) = %q, want %q", tt.url, got, tt.wantBVID)
			}
		})
	}
}

func TestExtractBVID_Invalid(t *testing.T) {
	urls := []string{
		"https://www.bilibili.com/video/av170001",
		"https://www.youtube.com/watch?v=abc123",
		"",
	}
	re := regexp.MustCompile(`BV[a-zA-Z0-9]+`)
	for _, u := range urls {
		if got := re.FindString(u); got != "" {
			t.Errorf("expected no match for %q, got %q", u, got)
		}
	}
}

func TestParseURL_Routing(t *testing.T) {
	// Create a mock Bilibili API server for offline testing.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{Code: 0, Message: "0"}
		data, _ := json.Marshal(VideoAPIResponse{
			BVID:  "BV1qt4y1X7TW",
			Title: "Test",
			Pages: []*PageInfo{{CID: 1, Page: 1}},
		})
		resp.Data = data
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	transport := &singleHostTransport{base: server.URL}
	authMgr := auth.NewAuthManager(t.TempDir(), logrus.New())
	p := &BilibiliParser{
		client:      &http.Client{Transport: transport},
		authManager: authMgr,
		logger:      logrus.New(),
	}

	// Valid video URL → should route to video type.
	info, err := p.ParseURL("https://www.bilibili.com/video/BV1qt4y1X7TW")
	if err != nil {
		t.Fatalf("ParseURL(video) error: %v", err)
	}
	if info.Type != "video" {
		t.Errorf("ParseURL(video).Type = %q, want video", info.Type)
	}

	// Unsupported URLs should return an error (routing or parsing).
	for _, u := range []string{"https://www.example.com/video/123", ""} {
		_, err := p.ParseURL(u)
		if err == nil {
			t.Errorf("ParseURL(%q) expected error, got nil", u)
		}
	}
}

func TestGetBestQualityStream(t *testing.T) {
	p := &BilibiliParser{}
	streams := []*StreamInfo{
		{Quality: 16, Resolution: "360p"},
		{Quality: 80, Resolution: "1080p"},
		{Quality: 64, Resolution: "720p"},
		{Quality: 32, Resolution: "480p"},
	}
	got := p.GetBestQualityStream(streams)
	if got == nil {
		t.Fatal("expected non-nil stream")
	}
	if got.Quality != 80 {
		t.Errorf("GetBestQualityStream quality = %d, want 80", got.Quality)
	}
}

func TestGetBestQualityStream_Empty(t *testing.T) {
	p := &BilibiliParser{}
	got := p.GetBestQualityStream(nil)
	if got != nil {
		t.Error("expected nil for empty streams")
	}
	got = p.GetBestQualityStream([]*StreamInfo{})
	if got != nil {
		t.Error("expected nil for empty streams")
	}
}

func TestGetStreamByQuality(t *testing.T) {
	p := &BilibiliParser{}
	streams := []*StreamInfo{
		{Quality: 16, Resolution: "360p"},
		{Quality: 64, Resolution: "720p"},
		{Quality: 80, Resolution: "1080p"},
	}

	tests := []struct {
		quality     string
		wantQuality int
	}{
		{"1080p", 80},
		{"720p", 64},
		{"360p", 16},
		{"best", 80},
		{"unknown", 80}, // falls back to best
	}
	for _, tt := range tests {
		t.Run(tt.quality, func(t *testing.T) {
			got := p.GetStreamByQuality(streams, tt.quality)
			if got == nil {
				t.Fatal("expected non-nil stream")
			}
			if got.Quality != tt.wantQuality {
				t.Errorf("GetStreamByQuality(%q) quality = %d, want %d", tt.quality, got.Quality, tt.wantQuality)
			}
		})
	}
}

func TestGetStreamByQuality_Empty(t *testing.T) {
	p := &BilibiliParser{}
	got := p.GetStreamByQuality(nil, "1080p")
	if got != nil {
		t.Error("expected nil for empty streams")
	}
}

func TestVideoAPIResponse_Unmarshal(t *testing.T) {
	body := `{
		"code": 0,
		"message": "0",
		"data": {
			"bvid": "BV1qt4y1X7TW",
			"aid": 123456,
			"title": "测试视频标题",
			"desc": "视频描述",
			"duration": 360,
			"pages": [
				{"cid": 111, "part": "Part 1", "duration": 180, "page": 1},
				{"cid": 222, "part": "Part 2", "duration": 180, "page": 2}
			]
		}
	}`

	var apiResp APIResponse
	if err := json.Unmarshal([]byte(body), &apiResp); err != nil {
		t.Fatal(err)
	}
	if apiResp.Code != 0 {
		t.Errorf("code = %d, want 0", apiResp.Code)
	}

	var videoData VideoAPIResponse
	if err := json.Unmarshal(apiResp.Data, &videoData); err != nil {
		t.Fatal(err)
	}
	if videoData.BVID != "BV1qt4y1X7TW" {
		t.Errorf("bvid = %q, want BV1qt4y1X7TW", videoData.BVID)
	}
	if videoData.Title != "测试视频标题" {
		t.Errorf("title = %q, want 测试视频标题", videoData.Title)
	}
	if len(videoData.Pages) != 2 {
		t.Errorf("pages len = %d, want 2", len(videoData.Pages))
	}
}

func TestPlaylistAPIResponse_Unmarshal(t *testing.T) {
	body := `{
		"code": 0,
		"message": "0",
		"data": {
			"title": "测试番剧",
			"episodes": [
				{"bvid": "BV001", "cid": 100, "title": "Ep1", "duration": 1440, "index": 1},
				{"bvid": "BV002", "cid": 200, "title": "Ep2", "duration": 1440, "index": 2}
			]
		}
	}`

	var apiResp APIResponse
	if err := json.Unmarshal([]byte(body), &apiResp); err != nil {
		t.Fatal(err)
	}
	if apiResp.Code != 0 {
		t.Errorf("code = %d, want 0", apiResp.Code)
	}

	var playlistData struct {
		Title    string `json:"title"`
		Episodes []struct {
			BVID     string `json:"bvid"`
			CID      int64  `json:"cid"`
			Title    string `json:"title"`
			Duration int    `json:"duration"`
			Index    int    `json:"index"`
		} `json:"episodes"`
	}
	if err := json.Unmarshal(apiResp.Data, &playlistData); err != nil {
		t.Fatal(err)
	}
	if playlistData.Title != "测试番剧" {
		t.Errorf("title = %q", playlistData.Title)
	}
	if len(playlistData.Episodes) != 2 {
		t.Errorf("episodes len = %d, want 2", len(playlistData.Episodes))
	}
}

func TestStreamResponse_Unmarshal(t *testing.T) {
	body := `{
		"code": 0,
		"data": {
			"dash": {
				"video": [
					{
						"id": 80,
						"baseUrl": "https://example.com/video.m4s",
						"backupUrl": [],
						"bandwidth": 5000000,
						"mimeType": "video/mp4",
						"codecs": "avc1.640028",
						"width": 1920,
						"height": 1080,
						"frameRate": "30"
					}
				],
				"audio": [
					{
						"id": 30280,
						"baseUrl": "https://example.com/audio.m4s",
						"backupUrl": [],
						"bandwidth": 320000,
						"mimeType": "audio/mp4",
						"codecs": "mp4a.40.2"
					}
				]
			},
			"accept_quality": [80, 64, 32, 16],
			"accept_description": ["1080P", "720P", "480P", "360P"]
		}
	}`

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Dash struct {
				Video []struct {
					ID        int      `json:"id"`
					BaseURL   string   `json:"baseUrl"`
					BackupURL []string `json:"backupUrl"`
					Bandwidth int      `json:"bandwidth"`
					MimeType  string   `json:"mimeType"`
					Codecs    string   `json:"codecs"`
					Width     int      `json:"width"`
					Height    int      `json:"height"`
				} `json:"video"`
				Audio []struct {
					ID        int      `json:"id"`
					BaseURL   string   `json:"baseUrl"`
					BackupURL []string `json:"backupUrl"`
					Bandwidth int      `json:"bandwidth"`
					MimeType  string   `json:"mimeType"`
					Codecs    string   `json:"codecs"`
				} `json:"audio"`
			} `json:"dash"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	if len(resp.Data.Dash.Video) != 1 {
		t.Errorf("video streams = %d, want 1", len(resp.Data.Dash.Video))
	}
	if resp.Data.Dash.Video[0].ID != 80 {
		t.Errorf("video quality id = %d, want 80", resp.Data.Dash.Video[0].ID)
	}
	if len(resp.Data.Dash.Audio) != 1 {
		t.Errorf("audio streams = %d, want 1", len(resp.Data.Dash.Audio))
	}
}

func TestGetVideoInfo_HTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{
			Code:    0,
			Message: "0",
		}
		videoData := VideoAPIResponse{
			BVID:     "BV1qt4y1X7TW",
			AID:      123456,
			Title:    "Test Video",
			Desc:     "Description",
			Duration: 360,
			Pages: []*PageInfo{
				{CID: 111, Part: "P1", Duration: 180, Page: 1},
			},
		}
		dataBytes, _ := json.Marshal(videoData)
		resp.Data = dataBytes
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Build a parser whose HTTP client only talks to the test server.
	// We override the API URL base by directly calling an internal method
	// through a custom transport that rewrites the host.
	transport := &singleHostTransport{base: server.URL}
	authMgr := auth.NewAuthManager(t.TempDir(), logrus.New())
	p := &BilibiliParser{
		client:      &http.Client{Transport: transport},
		authManager: authMgr,
		logger:      logrus.New(),
	}

	videoInfo, err := p.getVideoInfo("BV1qt4y1X7TW")
	if err != nil {
		t.Fatalf("getVideoInfo failed: %v", err)
	}
	if videoInfo.BVID != "BV1qt4y1X7TW" {
		t.Errorf("bvid = %q, want BV1qt4y1X7TW", videoInfo.BVID)
	}
	if videoInfo.Title != "Test Video" {
		t.Errorf("title = %q, want Test Video", videoInfo.Title)
	}
	if len(videoInfo.Pages) != 1 {
		t.Errorf("pages len = %d, want 1", len(videoInfo.Pages))
	}
}

// singleHostTransport rewrites all requests to a single base URL for testing.
type singleHostTransport struct {
	base string
}

func (t *singleHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to the test server, preserving path and query.
	u := t.base + req.URL.Path
	if req.URL.RawQuery != "" {
		u += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequest(req.Method, u, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}
