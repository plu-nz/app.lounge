package sources

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/stremio"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		s    stremio.Stream
		want Info
	}{
		{"torrentio", stremio.Stream{InfoHash: "abc", Name: "Torrentio\n4k HDR", Title: "Movie.2160p.WEB\n👤 120 💾 2.3 GB ⚙️ YTS"},
			Info{Height: 2160, SizeBytes: gb(2.3), Seeders: 120, HasSeeds: true, HDR: true, Transport: Torrent}},
		{"1080 direct", stremio.Stream{URL: "http://x/a.mkv", Title: "Movie 1080p 850 MB"},
			Info{Height: 1080, SizeBytes: 850 << 20}},
		{"hint size wins", stremio.Stream{URL: "http://x", Title: "720p 9 GB", BehaviorHints: stremio.StreamHints{VideoSize: 1 << 30}},
			Info{Height: 720, SizeBytes: 1 << 30}},
		{"nothing", stremio.Stream{URL: "http://x", Name: "Stream"}, Info{}},
		{"dimensions", stremio.Stream{URL: "http://x", Title: "1920x1080"}, Info{Height: 1080}},
		{"seeds word", stremio.Stream{InfoHash: "a", Title: "720p Seeds: 7"}, Info{Height: 720, Seeders: 7, HasSeeds: true, Transport: Torrent}},
		{"no seeders on direct", stremio.Stream{URL: "http://x", Title: "👤 5"}, Info{}},
		{"comma size", stremio.Stream{URL: "http://x", Title: "1,5 GiB"}, Info{SizeBytes: gb(1.5)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Parse(c.s); got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func gb(f float64) int64 { return int64(f * float64(1<<30)) }

func group(streams ...stremio.Stream) []stremio.StreamGroup {
	return []stremio.StreamGroup{{AddonID: "a", Streams: streams}}
}

func TestBest(t *testing.T) {
	uhd := stremio.Stream{InfoHash: "1", Title: "2160p 👤 50 💾 30 GB"}
	fhd := stremio.Stream{InfoHash: "2", Title: "1080p 👤 80 💾 4 GB"}
	fhdDirect := stremio.Stream{URL: "http://x/f", Title: "1080p 3 GB"}
	hd := stremio.Stream{URL: "http://x/h", Title: "720p 1 GB"}
	dead := stremio.Stream{InfoHash: "3", Title: "1080p 👤 1 💾 3 GB"}
	external := stremio.Stream{ExternalURL: "http://x", Title: "2160p"}

	cases := []struct {
		name    string
		streams []stremio.Stream
		rules   Rules
		want    int // index into streams, -1 for none
	}{
		{"best quality", []stremio.Stream{hd, fhd, uhd}, Rules{}, 2},
		{"cap at 1080", []stremio.Stream{hd, fhd, uhd}, Rules{MaxHeight: 1080}, 1},
		{"size cap", []stremio.Stream{uhd, fhd, hd}, Rules{MaxSizeGB: 5}, 1},
		{"prefer direct beats resolution", []stremio.Stream{uhd, fhdDirect}, Rules{Prefer: "direct"}, 1},
		{"prefer torrent", []stremio.Stream{fhdDirect, fhd}, Rules{Prefer: "torrent"}, 1},
		{"min seeders", []stremio.Stream{dead, hd}, Rules{MinSeeders: 5}, 1},
		{"unknown seeders kept", []stremio.Stream{{InfoHash: "9", Title: "1080p"}}, Rules{MinSeeders: 50}, 0},
		{"external only", []stremio.Stream{external}, Rules{}, -1},
		{"nothing passes", []stremio.Stream{uhd}, Rules{MaxHeight: 720}, -1},
		{"unknown resolution between 480 and 720", []stremio.Stream{{URL: "http://x/u"}, hd}, Rules{}, 1},
		{"tie keeps addon order", []stremio.Stream{{URL: "http://x/1"}, {URL: "http://x/2"}}, Rules{}, 0},
		{"seeders break ties", []stremio.Stream{dead, fhd}, Rules{}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ref, ok := Best(group(c.streams...), c.rules)
			if c.want < 0 {
				if ok {
					t.Fatalf("got %+v, want none", ref)
				}
				return
			}
			if !ok || ref.Stream != c.want {
				t.Fatalf("got %+v ok=%v, want stream %d", ref, ok, c.want)
			}
		})
	}
}

func TestRankSpansGroups(t *testing.T) {
	groups := []stremio.StreamGroup{
		{AddonID: "a", Streams: []stremio.Stream{{URL: "http://a", Title: "720p"}}},
		{AddonID: "b", Streams: []stremio.Stream{{URL: "http://b", Title: "1080p"}}},
	}
	got := Rank(groups, Rules{})
	if len(got) != 2 || got[0] != (Ref{1, 0}) {
		t.Fatalf("got %+v", got)
	}
}
