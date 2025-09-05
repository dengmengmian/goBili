package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"goBili/auth"

	"github.com/sirupsen/logrus"
)

// BilibiliParser handles parsing of Bilibili URLs and API responses
type BilibiliParser struct {
	client      *http.Client
	authManager *auth.AuthManager
	logger      *logrus.Logger
}

// VideoInfo represents information about a video
type VideoInfo struct {
	BVID     string         `json:"bvid"`
	AID      int64          `json:"aid"`
	Title    string         `json:"title"`
	Desc     string         `json:"desc"`
	Duration int            `json:"duration"`
	Type     string         `json:"type"` // "video" or "playlist"
	Episodes []*EpisodeInfo `json:"episodes,omitempty"`
	Pages    []*PageInfo    `json:"pages,omitempty"`
}

// EpisodeInfo represents information about an episode in a playlist
type EpisodeInfo struct {
	BVID     string `json:"bvid"`
	CID      int64  `json:"cid"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Index    int    `json:"index"`
}

// PageInfo represents information about a page in a multi-page video
type PageInfo struct {
	CID      int64  `json:"cid"`
	Part     string `json:"part"`  // 分P标题
	Title    string `json:"title"` // 保留兼容性
	Duration int    `json:"duration"`
	Page     int    `json:"page"`
}

// StreamInfo represents video stream information
type StreamInfo struct {
	Quality     int    `json:"quality"`
	Format      string `json:"format"`
	VideoURL    string `json:"video_url"`
	AudioURL    string `json:"audio_url"`
	VideoCodecs string `json:"video_codecs"`
	AudioCodecs string `json:"audio_codecs"`
	Bandwidth   int    `json:"bandwidth"`
	Resolution  string `json:"resolution"`
}

// APIResponse represents the structure of Bilibili API responses
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// VideoAPIResponse represents video API response data
type VideoAPIResponse struct {
	BVID     string      `json:"bvid"`
	AID      int64       `json:"aid"`
	Title    string      `json:"title"`
	Desc     string      `json:"desc"`
	Duration int         `json:"duration"`
	Pages    []*PageInfo `json:"pages"`
}

// PlaylistAPIResponse represents playlist API response data
type PlaylistAPIResponse struct {
	Title    string         `json:"title"`
	Episodes []*EpisodeInfo `json:"episodes"`
}

// NewBilibiliParser creates a new Bilibili parser
func NewBilibiliParser(authManager *auth.AuthManager, logger *logrus.Logger) *BilibiliParser {
	return &BilibiliParser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		authManager: authManager,
		logger:      logger,
	}
}

// ParseURL parses a Bilibili URL and returns video information
func (p *BilibiliParser) ParseURL(rawURL string) (*VideoInfo, error) {
	// Parse the URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	// Extract BVID or other identifiers from URL
	if strings.Contains(u.Path, "/video/") {
		return p.parseVideoURL(rawURL)
	} else if strings.Contains(u.Path, "/bangumi/play/") {
		return p.parsePlaylistURL(rawURL)
	}

	return nil, fmt.Errorf("unsupported URL format")
}

// parseVideoURL parses a single video URL
func (p *BilibiliParser) parseVideoURL(rawURL string) (*VideoInfo, error) {
	// Extract BVID from URL
	bvidRegex := regexp.MustCompile(`BV[a-zA-Z0-9]+`)
	bvid := bvidRegex.FindString(rawURL)
	if bvid == "" {
		return nil, fmt.Errorf("could not extract BVID from URL")
	}

	// Get video information from API
	videoInfo, err := p.getVideoInfo(bvid)
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %v", err)
	}

	// Check if this is a multi-part video (has multiple pages)
	if len(videoInfo.Pages) > 1 {
		videoInfo.Type = "playlist"
		// Convert pages to episodes for consistency
		videoInfo.Episodes = make([]*EpisodeInfo, len(videoInfo.Pages))
		for i, page := range videoInfo.Pages {
			// Use the original B站 page title (part field) if available, otherwise fallback to generated title
			episodeTitle := page.Part
			if episodeTitle == "" {
				episodeTitle = fmt.Sprintf("%s - P%d", videoInfo.Title, page.Page)
			}

			videoInfo.Episodes[i] = &EpisodeInfo{
				BVID:     videoInfo.BVID,
				CID:      page.CID,
				Title:    episodeTitle,
				Duration: page.Duration,
				Index:    page.Page,
			}
		}
	} else {
		videoInfo.Type = "video"
	}

	return videoInfo, nil
}

// parsePlaylistURL parses a playlist URL
func (p *BilibiliParser) parsePlaylistURL(rawURL string) (*VideoInfo, error) {
	// Extract season ID from URL
	seasonRegex := regexp.MustCompile(`ss(\d+)`)
	matches := seasonRegex.FindStringSubmatch(rawURL)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract season ID from URL")
	}

	seasonID := matches[1]

	// Get playlist information from API
	playlistInfo, err := p.getPlaylistInfo(seasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %v", err)
	}

	playlistInfo.Type = "playlist"
	return playlistInfo, nil
}

// getVideoInfo fetches video information from Bilibili API
func (p *BilibiliParser) getVideoInfo(bvid string) (*VideoInfo, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvid)

	req, err := p.authManager.CreateAuthenticatedRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	// Parse the data
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, err
	}

	var videoData VideoAPIResponse
	if err := json.Unmarshal(dataBytes, &videoData); err != nil {
		return nil, err
	}

	// Convert to VideoInfo
	videoInfo := &VideoInfo{
		BVID:     videoData.BVID,
		AID:      videoData.AID,
		Title:    videoData.Title,
		Desc:     videoData.Desc,
		Duration: videoData.Duration,
		Pages:    videoData.Pages,
	}

	return videoInfo, nil
}

// getPlaylistInfo fetches playlist information from Bilibili API
func (p *BilibiliParser) getPlaylistInfo(seasonID string) (*VideoInfo, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/pgc/view/web/season?season_id=%s", seasonID)

	req, err := p.authManager.CreateAuthenticatedRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
	}

	// Parse the data
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, err
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

	if err := json.Unmarshal(dataBytes, &playlistData); err != nil {
		return nil, err
	}

	// Convert to VideoInfo
	videoInfo := &VideoInfo{
		Title: playlistData.Title,
		Type:  "playlist",
	}

	// Convert episodes
	for _, ep := range playlistData.Episodes {
		episode := &EpisodeInfo{
			BVID:     ep.BVID,
			CID:      ep.CID,
			Title:    ep.Title,
			Duration: ep.Duration,
			Index:    ep.Index,
		}
		videoInfo.Episodes = append(videoInfo.Episodes, episode)
	}

	return videoInfo, nil
}

// GetVideoStreams fetches available video streams for a video
func (p *BilibiliParser) GetVideoStreams(videoInfo *VideoInfo) ([]*StreamInfo, error) {
	return p.GetVideoStreamsForPage(videoInfo, 1)
}

// GetVideoStreamsForPage gets video streams for a specific page
func (p *BilibiliParser) GetVideoStreamsForPage(videoInfo *VideoInfo, pageNum int) ([]*StreamInfo, error) {
	// Find the specific page
	var cid int64
	if len(videoInfo.Pages) > 0 {
		// If pageNum is specified, find that page
		if pageNum > 0 && pageNum <= len(videoInfo.Pages) {
			cid = videoInfo.Pages[pageNum-1].CID
		} else {
			// Default to first page
			cid = videoInfo.Pages[0].CID
		}
	} else {
		// If no pages, we need to get the CID from the video info
		// This would require an additional API call
		return nil, fmt.Errorf("no pages found for video")
	}

	return p.getVideoStreamsByCID(videoInfo.BVID, cid)
}

// getVideoStreamsByCID fetches video streams by CID
func (p *BilibiliParser) getVideoStreamsByCID(bvid string, cid int64) ([]*StreamInfo, error) {
	// Call the play URL API
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=0&fnval=16&fourk=1", bvid, cid)

	req, err := p.authManager.CreateAuthenticatedRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
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
					FrameRate string   `json:"frameRate"`
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
			AcceptQuality     []int    `json:"accept_quality"`
			AcceptDescription []string `json:"accept_description"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("failed to get video streams: %d", apiResp.Code)
	}

	// Convert to StreamInfo
	var streams []*StreamInfo

	// Quality mapping
	qualityMap := map[int]int{
		80: 80, // 1080p
		64: 64, // 720p
		32: 32, // 480p
		16: 16, // 360p
	}

	// Process video streams
	for _, video := range apiResp.Data.Dash.Video {
		quality, exists := qualityMap[video.ID]
		if !exists {
			continue
		}

		// Find corresponding audio stream
		var audioURL string
		if len(apiResp.Data.Dash.Audio) > 0 {
			audioURL = apiResp.Data.Dash.Audio[0].BaseURL
		}

		stream := &StreamInfo{
			Quality:     quality,
			Format:      "mp4",
			VideoURL:    video.BaseURL,
			AudioURL:    audioURL,
			VideoCodecs: video.Codecs,
			AudioCodecs: func() string {
				if len(apiResp.Data.Dash.Audio) > 0 {
					return apiResp.Data.Dash.Audio[0].Codecs
				}
				return ""
			}(),
			Bandwidth:  video.Bandwidth,
			Resolution: fmt.Sprintf("%dx%d", video.Width, video.Height),
		}

		streams = append(streams, stream)
	}

	// If no DASH streams, try legacy format
	if len(streams) == 0 {
		return p.getLegacyVideoStreams(bvid, cid)
	}

	return streams, nil
}

// getLegacyVideoStreams gets video streams in legacy format
func (p *BilibiliParser) getLegacyVideoStreams(bvid string, cid int64) ([]*StreamInfo, error) {
	apiURL := fmt.Sprintf("https://api.bilibili.com/x/player/playurl?bvid=%s&cid=%d&qn=80", bvid, cid)

	req, err := p.authManager.CreateAuthenticatedRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			DURL []struct {
				URL    string `json:"url"`
				Size   int64  `json:"size"`
				Length int    `json:"length"`
			} `json:"durl"`
			Quality int `json:"quality"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("failed to get legacy video streams: %d", apiResp.Code)
	}

	var streams []*StreamInfo
	for _, durl := range apiResp.Data.DURL {
		stream := &StreamInfo{
			Quality:     apiResp.Data.Quality,
			Format:      "flv",
			VideoURL:    durl.URL,
			AudioURL:    "", // Legacy format usually has combined video+audio
			VideoCodecs: "avc1",
			AudioCodecs: "mp4a",
			Bandwidth:   0,
			Resolution:  "unknown",
		}
		streams = append(streams, stream)
	}

	return streams, nil
}

// GetBestQualityStream returns the highest quality stream available
func (p *BilibiliParser) GetBestQualityStream(streams []*StreamInfo) *StreamInfo {
	if len(streams) == 0 {
		return nil
	}

	best := streams[0]
	for _, stream := range streams[1:] {
		if stream.Quality > best.Quality {
			best = stream
		}
	}

	return best
}

// GetStreamByQuality returns a stream with the specified quality
func (p *BilibiliParser) GetStreamByQuality(streams []*StreamInfo, quality string) *StreamInfo {
	qualityMap := map[string]int{
		"best":  80,
		"1080p": 80,
		"720p":  64,
		"480p":  32,
		"360p":  16,
	}

	targetQuality, exists := qualityMap[quality]
	if !exists {
		return p.GetBestQualityStream(streams)
	}

	for _, stream := range streams {
		if stream.Quality == targetQuality {
			return stream
		}
	}

	// If exact quality not found, return the best available
	return p.GetBestQualityStream(streams)
}
