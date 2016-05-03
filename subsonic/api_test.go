package subsonic

import (
	"encoding/json"
	"testing"
)

const (
	errMsg = "Something went wrong, blabla."
)

const (
	Jhead = `{"subsonic-response": {`

	Jtail = `
 "status": "ok",
 "xmlns": "http://subsonic.org/restapi",
 "version": "1.8.0"
}}`

	Jerr = `
  "error": {
  "message": "` + errMsg + `",
  "code": 40
 }`
)

var (
	buf interface{} // buffer for meta-tests (see :/Unmarshal/)
)

func TestPing(t *testing.T) {
	// successful case:
	j := []byte(Jhead + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	if err := parsePingResp(j); err != nil {
		t.Error("unexpected error:", err)
	}

	// error case:
	j = []byte(Jhead + Jerr + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	if err := parsePingResp(j); err != nil {
		if err.Error() != errMsg {
			t.Error("unexpected error:", err)
		}
	} else {
		t.Error("expected error found nil")
	}
}

func TestGetArtists(t *testing.T) {
	// common case:
	names := []string{"A1", "A2", "Kwyjibo"}
	d := `
 "artists": {"index": [
  {
   "name": "A",
   "artist": [
    {
     "id": 0,
     "name": "` + names[0] + `",
     "albumCount": 7
    },
    {
     "id": 1,
     "name": "` + names[1] + `",
     "coverArt": "ar-222",
     "albumCount": 1
    }
   ]
  },
  {
   "name": "K",
   "artist": {
     "id": 2,
     "name": "` + names[2] + `",
    "coverArt": "ar-23",
    "albumCount": 14
   }
  }
 ]}`
	j := []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err := parseGetArtistsResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(names) {
		t.Fatal(len(s), "≠", len(names))
	}
	for i, a := range s {
		if a.Id != i {
			t.Error(a.Id, "≠", i)
		}
		if a.Name != s[i].Name {
			t.Error(a.Name, "≠", names[i])
		}
	}

	// numbers in name:
	names = []string{"42", "0.12", "3.14"}
	d = `
 "artists": {"index": [
  {
   "name": "#",
   "artist": [
    {
     "id": 0,
     "name": 42,
     "albumCount": 7
    },
   {
     "id": 1,
     "name": 0.02,
     "albumCount": 7
    },
    {
     "id": 2,
     "name": 3.1415,
     "coverArt": "ar-222",
     "albumCount": 1
    }
  ]
  }
 ]}`
	j = []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err = parseGetArtistsResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(names) {
		t.Fatal(len(s), "≠", len(names))
	}
	for i, a := range s {
		if a.Id != i {
			t.Error(a.Id, "≠", i)
		}
		if a.Name != s[i].Name {
			t.Error(a.Name, "≠", names[i])
		}
	}

	// single element:
	name := "really poor library, mate"
	d = `
 "artists": {"index": {
   "name": "#",
   "artist": {
     "id": 0,
     "name": "` + name + `",
     "albumCount": 1
  }
 }}`
	j = []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err = parseGetArtistsResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 1 {
		t.Fatal(len(s), "≠", 1)
	}
	if s[0].Name != name {
		t.Error(s[0].Name, "≠", name)
	}

	// error case:
	j = []byte(Jhead + Jerr + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	if _, err := parseGetArtistsResp([]byte(j)); err != nil {
		if err.Error() != errMsg {
			t.Error("unexpected error:", err)
		}
	} else {
		t.Error("expected error found nil")
	}
}

func TestGetArtist(t *testing.T) {
	// single album
	name := "Dummy Disc"
	d := `
 "artist": {
  "id": 166,
  "album": {
   "id": 0,
   "duration": 3411,
   "songCount": 14,
   "created": "2013-03-18T12:21:33",
   "artistId": 166,
   "name": "` + name + `",
   "artist": "DummyArtist",
   "coverArt": "al-511"
  },
  "name": "DummyArtist",
  "coverArt": "ar-166",
  "albumCount": 1
 }`
	j := []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err := parseGetArtistResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if s[0].Id != 0 {
		t.Error("expected", 0, "found", s[0].Id)
	}
	if s[0].Name != name {
		t.Error("expected", name, "found", s[0].Name)
	}

	// multi albums
	names := []string{"Very Bad Disc", "Greatest Hits"}
	d = `
 "artist": {
  "id": 13,
  "album": [
   {
    "id": 0,
    "duration": 2517,
    "songCount": 10,
    "created": "2013-03-12T11:32:55",
    "artistId": 13,
    "name": "` + names[0] + `",
    "artist": "Rozzy"
   },
   {
    "id": 1,
    "duration": 2421,
    "songCount": 9,
    "created": "2013-03-12T11:33:43",
    "artistId": 13,
    "name": "` + names[1] + `",
    "artist": "Rozzy"
   }
  ],
  "name": "Rozzy",
  "albumCount": 2
 }`
	j = []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err = parseGetArtistResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(names) {
		t.Fatal(len(s), "≠", len(names))
	}
	for i, a := range s {
		if a.Name != names[i] {
			t.Error(a.Name, "≠", names[i])
		}
	}
}

func TestGetAlbum(t *testing.T) {
	songs := []Song{
		Song{Resource{1, "Track1"}, 1, "mp3"},
		Song{Resource{2, "Track2"}, 2, "ogg"},
	}
	d := `
 "album": {
  "id": 1,
  "song": [
   {
    "genre": 17,
    "albumId": 63,
    "album": "Dummy Disc",
    "track": 1,
    "parent": 805,
    "contentType": "audio/mpeg",
    "isDir": false,
    "type": "music",
    "suffix": "` + songs[0].Suffix + `",
    "isVideo": false,
    "size": 8308552,
    "id": 1,
    "title": "` + songs[0].Name + `",
    "duration": 207,
    "artistId": 13,
    "created": "2013-03-12T11:36:04",
    "path": "A/B/1.m3",
    "year": 1979,
    "artist": "Rozzy",
    "bitRate": 320
   },
   {
    "genre": 17,
    "albumId": 63,
    "album": "Dummy Disc",
    "track": 2,
    "parent": 805,
    "contentType": "audio/mpeg",
    "isDir": false,
    "type": "music",
     "suffix": "` + songs[1].Suffix + `",
    "isVideo": false,
    "size": 8308552,
    "id": 2,
    "title": "` + songs[1].Name + `",
    "duration": 376,
    "artistId": 13,
    "created": "2013-03-12T11:38:16",
    "path": "A/B/2.m3",
    "year": 1979,
    "artist": "Rozzy",
    "bitRate": 320
   }
  ],
  "duration": 2484,
  "songCount": 2,
  "created": "2013-03-12T11:37:46",
  "artistId": 13,
  "name": "Dummy Disc",
  "artist": "Rozzy"
 }`
	j := []byte(Jhead + d + "," + Jtail)
	if err := json.Unmarshal(j, &buf); err != nil {
		t.Fatal("EPIC FAIL: TEST IS BROKEN:", err)
	}
	s, err := parseGetAlbumResp(j)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != len(songs) {
		t.Fatal(len(s), "≠", len(songs))
	}
	for i, j := range s {
		if j.Id != songs[i].Id {
			t.Error(j.Id, "≠", songs[i].Id)
		}
		if j.Name != songs[i].Name {
			t.Error(j.Name, "≠", songs[i].Name)
		}
		if j.Number != songs[i].Number {
			t.Error(j.Number, "≠", songs[i].Number)
		}
		if j.Suffix != songs[i].Suffix {
			t.Error(j.Number, "≠", songs[i].Suffix)
		}
	}
}
