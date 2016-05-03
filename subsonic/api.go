package subsonic

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
)

func (c *Client) doReq(url string) ([]byte, error) {
	resp, err := c.cli.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

const (
	APIversion = "1.8.0"
	ClientName = "subsonicfs"
)

type Client struct {
	urlfmt string
	cli    *http.Client
}

func NewClient(host, user, password string, secure bool) *Client {
	var t http.Transport
	schema := "http"
	if secure {
		schema += "s"
		tc := tls.Config{InsecureSkipVerify: true} // FIXME
		t.TLSClientConfig = &tc
	}
	schema += "://"
	u := fmt.Sprintf("%s%s/rest/%%s.view?f=json&u=%s&p=%s&v=%s&c=%s",
		schema, host, user, password, APIversion, ClientName)
	return &Client{u, &http.Client{Transport: &t}}
}

type ReqError struct {
	Code    int
	Message string
}

func (e ReqError) Error() string {
	return e.Message
}

func parsePingResp(data []byte) error {
	var buf struct {
		R struct {
			Error *ReqError
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(data, &buf); err != nil {
		return err
	}
	if buf.R.Error != nil {
		return buf.R.Error
	}
	return nil
}

func (c *Client) Ping() error {
	url := fmt.Sprintf(c.urlfmt, "ping")
	resp, err := c.doReq(url)
	if err != nil {
		return err
	}
	return parsePingResp(resp)
}

type Resource struct {
	Id   int
	Name string
}

type Artist Resource

func parseArtistMap(m map[string]interface{}) (*Artist, error) {
	var a Artist
	v, ok := m["id"]
	if !ok {
		return nil, fmt.Errorf("field 'id' not found while decoding artist")
	}
	switch vv := v.(type) {
	case float64:
		a.Id = int(vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding artist: expecting float64", vv)
	}

	v, ok = m["name"]
	if !ok {
		return nil, fmt.Errorf("field 'name' not found while decoding artist")
	}
	switch vv := v.(type) {
	case string:
		a.Name = html.UnescapeString(vv)
	case float64:
		a.Name = fmt.Sprintf("%v", vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding artist: expecting string or float64", vv)
	}
	return &a, nil
}

func parseIndexMap(m map[string]interface{}) ([]Artist, error) {
	a, ok := m["artist"]
	if !ok {
		return nil, fmt.Errorf("field 'artist' not found while decoding index entry")
	}
	var retv []Artist
	switch aa := a.(type) {
	case map[string]interface{}:
		tmp, err := parseArtistMap(aa)
		if err != nil {
			return nil, err
		}
		retv = append(retv, *tmp)

	case []interface{}:
		for _, v := range aa {
			switch vv := v.(type) {
			case map[string]interface{}:
				tmp, err := parseArtistMap(vv)
				if err != nil {
					return nil, err
				}
				retv = append(retv, *tmp)

			default:
				return nil, fmt.Errorf("unexpected type (%T) while decoding artist array: expecting map[string]interface{}", vv)
			}
		}

	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding index entry: expecting map[string]interface{} or []interface{}", aa)
	}
	return retv, nil
}

func parseGetArtistsResp(data []byte) ([]Artist, error) {
	var buf struct {
		R struct {
			Error   *ReqError
			Artists struct {
				Index interface{}
			}
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(data, &buf); err != nil {
		return nil, err
	}
	if buf.R.Error != nil {
		return nil, buf.R.Error
	}
	var retv []Artist
	switch index := buf.R.Artists.Index.(type) {
	case map[string]interface{}:
		return parseIndexMap(index)

	case []interface{}:
		for _, v := range index {
			switch vv := v.(type) {
			case map[string]interface{}:
				tmp, err := parseIndexMap(vv)
				if err != nil {
					return nil, err
				}
				retv = append(retv, tmp...)

			default:
				return nil, fmt.Errorf("unexpected type (%T) while decoding index array: expecting map[string]interface{}", vv)
			}
		}

	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding index: expecting []interface{} or map[string]interface{}", index)
	}
	return retv, nil
}
func (c *Client) GetArtists() ([]Artist, error) {
	url := fmt.Sprintf(c.urlfmt, "getArtists")
	resp, err := c.doReq(url)
	if err != nil {
		return nil, err
	}
	return parseGetArtistsResp(resp)
}

type Album Resource

func parseAlbumMap(m map[string]interface{}) (*Album, error) {
	var a Album
	v, ok := m["id"]
	if !ok {
		return nil, fmt.Errorf("field 'id' not found while decoding album")
	}
	switch vv := v.(type) {
	case float64:
		a.Id = int(vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding album: expecting float64", vv)
	}

	v, ok = m["name"]
	if !ok {
		return nil, fmt.Errorf("field 'name' not found while decoding album")
	}
	switch vv := v.(type) {
	case string:
		a.Name = html.UnescapeString(vv)
	case float64:
		a.Name = fmt.Sprintf("%v", vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding album: expecting string or float64", vv)
	}
	return &a, nil
}

func parseGetArtistResp(data []byte) ([]Album, error) {
	var buf struct {
		R struct {
			Error  *ReqError
			Artist struct {
				Album interface{}
			}
		} `json:"subsonic-response"`
	}
	if err := json.Unmarshal(data, &buf); err != nil {
		return nil, err
	}
	if buf.R.Error != nil {
		return nil, buf.R.Error
	}
	var retv []Album
	switch a := buf.R.Artist.Album.(type) {
	case map[string]interface{}:
		tmp, err := parseAlbumMap(a)
		if err != nil {
			return nil, err
		}
		retv = append(retv, *tmp)

	case []interface{}:
		for _, v := range a {
			switch vv := v.(type) {
			case map[string]interface{}:
				tmp, err := parseAlbumMap(vv)
				if err != nil {
					return nil, err
				}
				retv = append(retv, *tmp)

			default:
				return nil, fmt.Errorf("unexpected type (%T) while decoding album array: expecting map[string]interface{}", vv)
			}
		}

	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding artist entry: expecting map[string]interface{} or []interface{}", a)
	}
	return retv, nil
}

func (c *Client) GetArtist(artist int) ([]Album, error) {
	url := fmt.Sprintf(c.urlfmt+"&id=%d", "getArtist", artist)
	resp, err := c.doReq(url)
	if err != nil {
		return nil, err
	}
	return parseGetArtistResp(resp)
}

type Song struct {
	Resource
	Number int
	Suffix string
}

func parseSongMap(m map[string]interface{}) (*Song, error) {
	var s Song
	v, ok := m["id"]
	if !ok {
		return nil, fmt.Errorf("field 'id' not found while decoding song")
	}
	switch vv := v.(type) {
	case float64:
		s.Id = int(vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding song: expecting float64", vv)
	}

	v, ok = m["title"]
	if !ok {
		return nil, fmt.Errorf("field 'title' not found while decoding song")
	}
	switch vv := v.(type) {
	case string:
		s.Name = html.UnescapeString(vv)
	case float64:
		s.Name = fmt.Sprintf("%v", vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding song: expecting string", vv)
	}

	v, ok = m["track"]
	if !ok {
		return nil, fmt.Errorf("field 'track' not found while decoding song")
	}
	switch vv := v.(type) {
	case float64:
		s.Number = int(vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding album: expecting float64", vv)
	}

	v, ok = m["suffix"]
	if !ok {
		return nil, fmt.Errorf("field 'suffix' not found while decoding song")
	}
	switch vv := v.(type) {
	case string:
		s.Suffix = html.UnescapeString(vv)
	case float64:
		s.Suffix = fmt.Sprintf("%v", vv)
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding album: expecting string or float64", vv)
	}
	return &s, nil
}

func parseGetAlbumResp(data []byte) ([]Song, error) {
	var buf struct {
		R struct {
			Error *ReqError
			Album struct {
				Song interface{}
			}
		} `json:"subsonic-response"`
	}
	if buf.R.Error != nil {
		return nil, buf.R.Error
	}
	if err := json.Unmarshal(data, &buf); err != nil {
		return nil, err
	}
	var retv []Song
	switch s := buf.R.Album.Song.(type) {
	case map[string]interface{}:
		tmp, err := parseSongMap(s)
		if err != nil {
			return nil, err
		}
		retv = append(retv, *tmp)

	case []interface{}:
		for _, v := range s {
			switch vv := v.(type) {
			case map[string]interface{}:
				tmp, err := parseSongMap(vv)
				if err != nil {
					return nil, err
				}
				retv = append(retv, *tmp)

			default:
				return nil, fmt.Errorf("unexpected type (%T) while decoding song array: expecting map[string]interface{}", vv)
			}
		}
	default:
		return nil, fmt.Errorf("unexpected type (%T) while decoding album entry: expecting map[string]interface{} or []interface{}", s)
	}
	return retv, nil

}

func (c *Client) GetAlbum(album int) ([]Song, error) {
	url := fmt.Sprintf(c.urlfmt+"&id=%d", "getAlbum", album)
	resp, err := c.doReq(url)
	if err != nil {
		return nil, err
	}
	return parseGetAlbumResp(resp)
}

func (c *Client) Stream(song, maxbitrate int) (io.ReadCloser, error) {
	url := fmt.Sprintf(c.urlfmt+"&id=%d&maxBitRate=%d", "stream", song, maxbitrate)
	resp, err := c.cli.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
