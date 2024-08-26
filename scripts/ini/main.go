package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hrfee/jfa-go/common"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v3"
)

func fixDescription(desc string) string {
	return "; " + strings.ReplaceAll(desc, "\n", "\n; ")
}

func generateIni(yamlPath string, iniPath string) {
	yamlFile, err := os.ReadFile(yamlPath)
	if err != nil {
		panic(err)
	}
	configBase := common.Config{}
	err = yaml.Unmarshal(yamlFile, &configBase)
	if err != nil {
		panic(err)
	}
	conf := ini.Empty()

	for _, section := range configBase.Sections {
		cSection, err := conf.NewSection(section.Section)
		if err != nil {
			panic(err)
		}
		if section.Meta.Description != "" {
			cSection.Comment = fixDescription(section.Meta.Description)
		}
		for _, setting := range section.Settings {
			if setting.Type == common.NoteType {
				continue
			}
			val := ""
			if setting.Value != nil {
				// Easy way to convert bools and numbers to strings,
				// Instead of checking setting.Type
				val = fmt.Sprintf("%v", setting.Value)
			}
			cKey, err := cSection.NewKey(setting.Setting, val)
			if err != nil {
				panic(err)
			}
			if setting.Description != "" {
				cKey.Comment = fixDescription(setting.Description)
			}
			// Explain how to use list type
			if setting.Type == common.ListType {
				if cKey.Comment != "" {
					cKey.Comment += "\n"
				}
				cKey.Comment += `List type: duplicate and edit the line to add more entries.`
			}
		}
	}

	err = conf.SaveTo(iniPath)
	if err != nil {
		panic(err)
	}
}

// Compares two inis, used to check this script does the equivalent of the old generate_ini.py.
func compareInis(p1, p2 string) {
	cA, err := ini.ShadowLoad(p1)
	if err != nil {
		panic(err)
	}

	cB, err := ini.ShadowLoad(p2)
	if err != nil {
		panic(err)
	}

	for _, pair := range [][2]*ini.File{{cA, cB}, {cB, cA}} {
		s1 := pair[0].Sections()
		s2 := pair[1].Sections()
		for i := range s1 {
			if s1[i].Name() != s2[i].Name() {
				panic(fmt.Errorf("mismatching section order: s0[i]=%s, s1[i]=%s", s1[i].Name(), s2[i].Name()))
			}
			// fmt.Println("Section order matches")
			st1 := s1[i].Keys()
			st2 := s2[i].Keys()
			for i := range st1 {
				if st1[i].Name() != st2[i].Name() {
					panic(fmt.Errorf("mismatching setting order: st1[i]=%s, st2[i]=%s", st1[i].Name(), st2[i].Name()))
				}
				if st1[i].Value() != st2[i].Value() {
					panic(fmt.Errorf("mismatching setting values: st1[i]=%s, st2[i]=%s", st1[i].Value(), st2[i].Value()))
				}
				// fmt.Println("Setting matches")
			}
		}
	}
}

func main() {
	var yamlPath string
	var iniPath string
	var comparePath string
	flag.StringVar(&yamlPath, "in", "", "Input of the config base in yaml.")
	flag.StringVar(&iniPath, "out", "", "Output path of an ini file.")
	flag.StringVar(&comparePath, "comp", "", "Path to ini file to compare against.")

	flag.Parse()

	if yamlPath == "" {
		panic(errors.New("invalid yaml path"))
	}
	if iniPath == "" {
		panic(errors.New("invalid ini path"))
	}

	generateIni(yamlPath, iniPath)

	if comparePath != "" {
		compareInis(iniPath, comparePath)
		fmt.Println("Passed.")
	}
}
