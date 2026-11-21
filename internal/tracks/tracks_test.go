package tracks

import (
	"testing"

	"github.com/plu-nz/app.lounge/internal/store"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"por": "pt", "pt-BR": "pt", "Portuguese": "pt", "pob": "pt", "pt": "pt",
		"eng": "en", "English": "en", "en_US": "en", "ger": "de", "deu": "de",
		"und": "", "": "", " ": "", "xx-YY": "xx", "Español": "es",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	if Same("", "") {
		t.Error("empty languages must not match")
	}
}

func tr(id int, typ, lang, title string, flags ...string) Track {
	t := Track{ID: id, Type: typ, Lang: lang, Title: title}
	for _, f := range flags {
		switch f {
		case "default":
			t.Default = true
		case "forced":
			t.Forced = true
		case "ext":
			t.External = true
		}
	}
	return t
}

func TestSelect(t *testing.T) {
	jpnEng := []Track{
		tr(1, "video", "", ""),
		tr(2, "audio", "jpn", "", "default"),
		tr(3, "audio", "eng", ""),
		tr(4, "sub", "eng", "Signs & Songs", "forced"),
		tr(5, "sub", "eng", "English SDH"),
		tr(6, "sub", "eng", "English"),
		tr(7, "sub", "por", "Portuguese (Brazil)"),
	}
	cases := []struct {
		name  string
		list  []Track
		prefs Prefs
		over  *Override
		want  Result
	}{
		{"no preferences", jpnEng, Prefs{}, nil, Result{}},
		{"audio by 639-2", jpnEng, Prefs{AudioLang: "eng"}, nil, Result{AudioID: 3}},
		{"audio by name", jpnEng, Prefs{AudioLang: "Japanese"}, nil, Result{AudioID: 2}},
		{"audio missing keeps default", jpnEng, Prefs{AudioLang: "fra"}, nil, Result{}},
		{"sub skips forced and SDH", jpnEng, Prefs{SubLang: "eng"}, nil, Result{SubID: 6}},
		{"sub by 639-1", jpnEng, Prefs{SubLang: "pt"}, nil, Result{SubID: 7}},
		{"sub off", jpnEng, Prefs{SubLang: "eng", SubMode: ModeOff}, nil, Result{SubOff: true}},
		{"auto audio differs", jpnEng, Prefs{AudioLang: "por", SubLang: "por", SubMode: ModeAuto}, nil, Result{SubID: 7}},
		{"auto audio same", jpnEng, Prefs{AudioLang: "eng", SubLang: "eng", SubMode: ModeAuto}, nil, Result{AudioID: 3, SubOff: true}},
		{"auto default audio in other language", jpnEng, Prefs{AudioLang: "fra", SubLang: "eng", SubMode: ModeAuto}, nil, Result{SubID: 6}},
		{"auto audio differs, no sub match", jpnEng, Prefs{AudioLang: "fra", SubLang: "ita", SubMode: ModeAuto}, nil, Result{}},
		{"override audio beats global", jpnEng, Prefs{AudioLang: "eng"}, &Override{Audio: &Choice{Lang: "jpn"}}, Result{AudioID: 2}},
		{"override sub off beats global", jpnEng, Prefs{SubLang: "eng"}, &Override{Sub: &Choice{Off: true}}, Result{SubOff: true}},
		{"override sub beats auto", jpnEng, Prefs{AudioLang: "eng", SubLang: "por", SubMode: ModeAuto}, &Override{Sub: &Choice{Lang: "pt"}}, Result{AudioID: 3, SubID: 7}},
		{"override without match falls back", jpnEng, Prefs{SubLang: "eng"}, &Override{Sub: &Choice{Lang: "fra"}}, Result{SubID: 6}},
		{"SDH before forced", []Track{tr(1, "sub", "eng", "Forced", "forced"), tr(2, "sub", "eng", "SDH")}, Prefs{SubLang: "eng"}, nil, Result{SubID: 2}},
		{"only forced is used", []Track{tr(1, "sub", "eng", "", "forced")}, Prefs{SubLang: "eng"}, nil, Result{SubID: 1}},
		{"embedded beats addon", []Track{tr(1, "sub", "eng", "x", "ext"), tr(2, "sub", "eng", "")}, Prefs{SubLang: "eng"}, nil, Result{SubID: 2}},
		{"addon used when no embedded match", []Track{tr(1, "sub", "spa", ""), tr(2, "sub", "eng", "eng", "ext")}, Prefs{SubLang: "eng"}, nil, Result{SubID: 2}},
		{"regular addon beats embedded SDH", []Track{tr(1, "sub", "eng", "SDH"), tr(2, "sub", "eng", "eng", "ext")}, Prefs{SubLang: "eng"}, nil, Result{SubID: 2}},
		{"commentary avoided", []Track{tr(1, "audio", "eng", "Director Commentary", "default"), tr(2, "audio", "eng", "")}, Prefs{AudioLang: "eng"}, nil, Result{AudioID: 2}},
		{"lowest id among equals", []Track{tr(3, "audio", "eng", ""), tr(2, "audio", "eng", "")}, Prefs{AudioLang: "eng"}, nil, Result{AudioID: 2}},
		{"empty list", nil, Prefs{AudioLang: "eng", SubLang: "eng"}, nil, Result{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Select(c.list, c.prefs, c.over); got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestOverridesPersist(t *testing.T) {
	st, err := store.OpenDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	o, _ := LoadOverrides(st)
	if o.Get("tt1") != nil {
		t.Fatal("expected no override")
	}
	_ = o.Remember("tt1", "audio", Choice{Lang: "jpn"})
	_ = o.Remember("tt1", "sub", Choice{Off: true})
	_ = o.Remember("", "sub", Choice{Off: true})
	o2, _ := LoadOverrides(st)
	got := o2.Get("tt1")
	if got == nil || got.Audio.Lang != "jpn" || !got.Sub.Off {
		t.Fatalf("got %+v", got)
	}
}
