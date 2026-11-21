package tracks

import "strings"

// languages maps every known spelling of a language to one key, its ISO
// 639-1 code. Each row lists the 639-1 code, the 639-2 codes, and names.
var languages = map[string]string{}

func init() {
	rows := [][]string{
		{"en", "eng", "english", "inglés", "ingles", "inglês"},
		{"pt", "por", "pob", "portuguese", "português", "portugues", "brazilian", "brazilian portuguese", "português do brasil"},
		{"es", "spa", "spl", "spanish", "español", "espanol", "castilian", "latino", "castellano"},
		{"fr", "fra", "fre", "french", "français", "francais"},
		{"de", "deu", "ger", "german", "deutsch"},
		{"it", "ita", "italian", "italiano"},
		{"ja", "jpn", "japanese", "日本語"},
		{"ko", "kor", "korean", "한국어"},
		{"ru", "rus", "russian", "русский"},
		{"zh", "zho", "chi", "chinese", "mandarin", "cantonese", "中文"},
		{"ar", "ara", "arabic"},
		{"hi", "hin", "hindi"},
		{"nl", "nld", "dut", "dutch"},
		{"pl", "pol", "polish"},
		{"tr", "tur", "turkish"},
		{"sv", "swe", "swedish"},
		{"da", "dan", "danish"},
		{"no", "nor", "nob", "norwegian"},
		{"fi", "fin", "finnish"},
		{"cs", "ces", "cze", "czech"},
		{"el", "ell", "gre", "greek"},
		{"he", "heb", "hebrew"},
		{"hu", "hun", "hungarian"},
		{"ro", "ron", "rum", "romanian"},
		{"uk", "ukr", "ukrainian"},
		{"th", "tha", "thai"},
		{"vi", "vie", "vietnamese"},
		{"id", "ind", "indonesian"},
	}
	for _, row := range rows {
		for _, name := range row {
			languages[name] = row[0]
		}
	}
}

// Normalize returns the ISO 639-1 key for a language code or name, such as
// "por", "pt-BR", or "Portuguese". Unknown values come back lowercased with
// any region removed, so equal unknown codes still match. Empty and
// undetermined codes ("und", "mis", "zxx") return "".
func Normalize(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	switch l {
	case "", "und", "mis", "mul", "zxx", "unknown":
		return ""
	}
	if k, ok := languages[l]; ok {
		return k
	}
	if i := strings.IndexAny(l, "-_"); i > 0 {
		if k, ok := languages[l[:i]]; ok {
			return k
		}
		return l[:i]
	}
	return l
}

// Same reports whether two language spellings name the same language.
func Same(a, b string) bool {
	na, nb := Normalize(a), Normalize(b)
	return na != "" && na == nb
}
