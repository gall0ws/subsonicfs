package main

import (
	"bitbucket.org/gall0ws/subsonicfs/subsonic"
	"code.google.com/p/go9p/p"
	"code.google.com/p/go9p/p/srv"

	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	maxbps = flag.Int("b", 192, "max bps")
	addr   = flag.String("l", ":5640", "listening network address")
	host   = flag.String("h", "", "subsonic server (e.g.: ss.example.com:1234)")
	tls    = flag.Bool("s", false, "enable http secure")
	passwd = flag.String("p", "", "subsonic password")
	user   = flag.String("u", "", "subsonic username")

	client  *subsonic.Client
	streams = struct {
		sync.Mutex
		m map[*srv.Fid]io.ReadCloser
	}{m: make(map[*srv.Fid]io.ReadCloser)}
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("subsonicfs: ")
}

func main() {
	flag.Parse()
	if *user == "" || *host == "" {
		flag.Usage()
		return
	}
	client = subsonic.NewClient(*host, *user, *passwd, *tls)
	if err := client.Ping(); err != nil {
		log.Fatalln(err)
		return
	}
	fs, err := buildFs()
	if err != nil {
		log.Fatalln(err)
	}
	fs.Start(fs)
	if err := fs.StartNetListener("tcp", *addr); err != nil {
		log.Fatalln(err)
	}
}

var (
	dirperm = uint32(p.DMDIR | 0555)
	owner   = p.OsUsers.Uid2User(os.Getuid())
	srepl   = strings.NewReplacer(
		`"`, "_",
		" ", "␣",
		"/", "_", // mandatory
		"'", "_",
		"(", "_",
		")", "_",
		"#", "_",
		"&", "and",
	)
)

func tr(s string) string {
	return srepl.Replace(strings.ToLower(s))
}

func buildFs() (*srv.Fsrv, error) {
	root := &srv.File{}
	if err := root.Add(nil, "/", owner, nil, dirperm, nil); err != nil {
		return nil, err
	}
	ctl := &Ctl{}
	if err := ctl.Add(root, "ctl", owner, nil, 0664, ctl); err != nil {
		return nil, err
	}

	artists, err := client.GetArtists()
	if err != nil {
		return nil, err
	}
	for _, artist := range artists {
		name := tr(artist.Name)
		r := []rune(name)[0]
		letter := string(r)
		if r < 'a' || r > 'z' {
			letter = "@" // subsonic uses '#', but I don't like it. 
		}
		index := root.Find(letter)
		if index == nil {
			index = &srv.File{}
			if err := index.Add(root, letter, owner, nil, dirperm, nil); err != nil {
				log.Printf("could not add index directory `%c': %s\n", index, err)
				continue
			}
		}
		dir := &ArtistDir{id: artist.Id}
		if err := dir.Add(index, name, owner, nil, dirperm, dir); err != nil {
			log.Printf("could not add artist directory `%s': %s\n", name, err)
			continue
		}
	}
	return srv.NewFileSrv(root), nil
}

type ArtistDir struct {
	srv.File
	sync.Once
	id int
}

func (d *ArtistDir) Stat(fid *srv.FFid) (e error) {
	f := func() {
		albums, err := client.GetArtist(d.id)
		if err != nil {
			log.Printf("could not load albums for artist %d: %s\n", d.id, err)
			e = err
		}
		for _, album := range albums {
			name := tr(album.Name)
			subdir := &AlbumDir{id: album.Id}
			if err := subdir.Add(&d.File, name, owner, nil, dirperm, subdir); err != nil {
				log.Printf("could not add subdirectory `%s': %s\n", name, err)
				continue
			}
		}
	}
	d.Do(f) // just once
	return
}

type AlbumDir struct {
	srv.File
	sync.Once
	id int
}

func (d *AlbumDir) Stat(fid *srv.FFid) (e error) {
	f := func() {
		songs, err := client.GetAlbum(d.id)
		if err != nil {
			e = err
		}
		for _, s := range songs {
			f := &SongFile{id: s.Id}
			name := tr(fmt.Sprintf("%02d_%s.%s", s.Number, s.Name, s.Suffix))
			if err := f.Add(&d.File, name, owner, nil, 0444, f); err != nil {
				e = err
			}
		}
	}
	d.Do(f) // just once
	return
}

type SongFile struct {
	srv.File
	id int
}

func (f *SongFile) Read(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	streams.Lock()
	defer streams.Unlock()
	src, ok := streams.m[fid.Fid]
	if !ok {
		if offset > 0 {
			return 0, nil
		}
		r, err := client.Stream(f.id, *maxbps)
		if err != nil {
			return 0, err
		}
		streams.m[fid.Fid] = r
		src = r
	}
	c, err := src.Read(buf)
	if err != nil {
		src.Close()
		delete(streams.m, fid.Fid)
		if err == io.EOF {
			return 0, nil
		}
		return c, err
	}
	return c, nil
}

func (f *SongFile) Clunk(fid *srv.FFid) error {
	streams.Lock()
	defer streams.Unlock()
	if src, ok := streams.m[fid.Fid]; ok {
		src.Close()
		delete(streams.m, fid.Fid)
	}
	return nil
}

type Ctl struct {
	srv.File
}

func (*Ctl) Read(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	return 0, nil
}

func (*Ctl) Write(fid *srv.FFid, data []byte, offset uint64) (int, error) {
	switch s := string(data); s {
	case "close":
		defer os.Exit(0)
	}
	return len(data), nil
}

/*
TODO
type MsgFile struct {
	srv.File
}

func (f *MsgFile) Read(fid *srv.FFid, buf []byte, offset uint64) (int, error) {
	// TODO
	return 0, nil
}

func (*MsgFile) Write(fid *srv.FFid, data []byte, offset uint64) (int, error) {
	// TODO
	return 0, nil
}
*/
