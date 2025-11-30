package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/hrfee/jfa-go/common"
)

func flattenOrder(c common.Config) (sections []string) {
	var traverseGroup func(groupName string) []string
	traverseGroup = func(groupName string) []string {
		out := []string{}
		for _, group := range c.Groups {
			if group.Group == groupName {
				for _, groupMember := range group.Members {
					if groupMember.Group != "" {
						out = append(out, traverseGroup(groupMember.Group)...)
					} else if groupMember.Section != "" {
						out = append(out, groupMember.Section)
					}
				}
				break
			}
		}
		return out
	}
	sections = make([]string, 0, len(c.Sections))
	for _, member := range c.Order {
		if member.Group != "" {
			sections = append(sections, traverseGroup(member.Group)...)
		} else if member.Section != "" {
			sections = append(sections, member.Section)
		}
	}
	return
}

func validateOrderCompleteness(c common.Config, sectOrder []string) (missing []string) {
	listedSects := map[string]bool{}
	for _, sect := range sectOrder {
		listedSects[sect] = true
	}

	for _, section := range c.Sections {
		if _, ok := listedSects[section.Section]; !ok {
			missing = append(missing, section.Section)
		}
	}
	return missing
}

func main() {
	var inPath string
	var outPath string
	flag.StringVar(&inPath, "in", "", "Input of the config base in yaml.")
	flag.StringVar(&outPath, "out", "", "Output of the checked and processed")

	flag.Parse()

	if inPath == "" {
		panic(errors.New("invalid input path"))
	}
	if outPath == "" {
		panic(errors.New("invalid output path"))
	}

	yamlFile, err := os.ReadFile(inPath)
	if err != nil {
		panic(err)
	}
	info, err := os.Stat(inPath)
	if err != nil {
		panic(err)
	}

	configBase := common.Config{}
	err = yaml.Unmarshal(yamlFile, &configBase)
	if err != nil {
		panic(err)
	}

	red := color.New(color.FgRed)

	if len(configBase.Order) > 0 {
		sectOrder := flattenOrder(configBase)
		missing := validateOrderCompleteness(configBase, sectOrder)
		if len(missing) > 0 {
			red.Fprintln(os.Stderr, "ERROR: Root order specified but the following sections were not listed, directly or indirectly:")
			for _, section := range missing {
				red.Fprintln(os.Stderr, "\t"+section)
			}
			os.Exit(1)
		}

		sectionMap := map[string]common.Section{}
		for _, sect := range configBase.Sections {
			sectionMap[sect.Section] = sect
		}

		for i, sect := range sectOrder {
			configBase.Sections[i] = sectionMap[sect]
		}

		fmt.Println("Re-ordered sections to follow root order.")
	}

	bytes, err := yaml.Marshal(&configBase)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(outPath, bytes, info.Mode())
	if err != nil {
		panic(err)
	}
}
