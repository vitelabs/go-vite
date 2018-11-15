package hd_bip

import (
	"github.com/tyler-smith/go-bip39/wordlists"
)

func GetWordList(lang string) []string {
	switch lang {
	case "en":
		return wordlists.English
	case "zh-Hans":
		return wordlists.ChineseSimplified
	case "zh-Hant":
		return wordlists.ChineseTraditional
	case "fr":
		return wordlists.French
	case "ja":
		return wordlists.Japanese
	case "it":
		return wordlists.Italian
	case "ko":
		return wordlists.Korean
	case "es":
		return wordlists.Spanish
	}
	return wordlists.English

}
