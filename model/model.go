package model

import (
	"fmt"
	"strings"
)

type Properties map[string][]string

func (p Properties) String() string {
	res := []string{}
	for key, vals := range p {
		res = append(res, fmt.Sprintf("%s=%s", key, strings.Join(vals, ",")))
	}
	return strings.Join(res, ";")
}

func (p Properties) Merge(other Properties) {
	for key, vals := range other {
		cur, ok := p[key]
		if !ok {
			cur = []string{}
		}
		uniq := map[string]bool{}
		for _, val := range cur {
			uniq[val] = true
		}
		for _, val := range vals {
			_, present := uniq[val]
			if !present {
				cur = append(cur, val)
			}
		}
		p[key] = cur
	}
}

type Version struct {
	Version string `json:"version"`
	File    string `json:"file"`
}

type Metadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Source struct {
	Url        string     `json:"url"`
	Repository string     `json:"repository"`
	Filter     string     `json:"filter"`
	User       string     `json:"user"`
	Password   string     `json:"password"`
	ApiKey     string     `json:"apiKey"`
	SshKey     string     `json:"ssh_key"`
	LogLevel   string     `json:"log_level"`
	CACert     string     `json:"ca_cert"`
	Threads    int        `json:"threads"`
	Props      Properties `json:"props"`
}

func (Source) Default() Source {
	return Source{
		Filter:   ".*",
		Threads:  3,
		Props:    Properties{},
		LogLevel: "ERROR",
	}
}

type InParams struct {
	MinSplit      int    `json:"min_split"`
	SplitCount    int    `json:"split_count"`
	Destination   string `json:"destination"`
	PropsFilename string `json:"props_filename"`
}

func (InParams) Default() InParams {
	return InParams{
		MinSplit:    5120,
		SplitCount:  3,
		Destination: ".",
	}
}

type OutParams struct {
	Directory     string     `json:"directory"`
	Props         Properties `json:"props"`
	PropsFilename string     `json:"props_filename"`
}

func (OutParams) Default() OutParams {
	return OutParams{
		Props: Properties{},
	}
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
}

func (CheckRequest) Default() CheckRequest {
	return CheckRequest{
		Source: Source{}.Default(),
	}
}

type InRequest struct {
	Source  Source   `json:"source"`
	Version Version  `json:"version"`
	Params  InParams `json:"params"`
}

func (InRequest) Default() InRequest {
	return InRequest{
		Source: Source{}.Default(),
		Params: InParams{}.Default(),
	}
}

type OutRequest struct {
	Source Source    `json:"source"`
	Params OutParams `json:"params"`
}

func (OutRequest) Default() OutRequest {
	return OutRequest{
		Source: Source{}.Default(),
		Params: OutParams{}.Default(),
	}
}

type Response struct {
	Metadata []Metadata `json:"metadata"`
	Version  Version    `json:"version"`
}
