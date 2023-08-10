package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sort"

	"regexp"

	"github.com/hashicorp/go-version"
)

const varsTemplate = `go_versions:
{{- range . }}
  - "{{ . }}"
{{- end }}
go_version_target: "{{ (index . 0) }}"
go_tarball: "{{"{{"}} go_version_target }}.linux-amd64.tar.gz{{"}}"}}"
go_download_location: "https://golang.google.cn/dl/{{"{{"}} go_tarball {{"}}"}}"
`

func generateAnsibleVars(goVersions []string, outputPath string) error {
	tmpl := template.Must(template.New("ansibleVars").Parse(varsTemplate))

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, goVersions)
	return err
}

func main() {
	url := "https://golang.google.cn/dl/"
	goVersions, err := getGoVersions(url)
	if err != nil {
		fmt.Println("Error fetching Go versions:", err)
		os.Exit(1)
	}
	err = generateAnsibleVars(goVersions, "../defaults/main.yml")
	if err != nil {
		fmt.Printf("Error generating Ansible vars file: %s\n", err)
		os.Exit(1)
	}
}

func getGoVersions(url string) ([]string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	goVersions := extractGoVersions(string(body))
	return goVersions, nil
}

func extractGoVersions(body string) []string {
	versionRegex := regexp.MustCompile(`go([\d.]+)\.linux-amd64\.tar\.gz`)
	versionStrings := versionRegex.FindAllString(body, -1)

	versions := make([]*version.Version, 0)

	for _, v := range versionStrings {
		versionNumber := versionRegex.FindStringSubmatch(v)[1]
		parsedVersion, err := version.NewVersion(versionNumber)
		if err != nil {
			fmt.Printf("Failed to parse version %s: %s\n", versionNumber, err)
			continue
		}
		versions = append(versions, parsedVersion)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[j].LessThan(versions[i])
	})

	uniqueVersions := uniqueVersionSlice(versions)

	numVersions := 20
	if len(uniqueVersions) < numVersions {
		numVersions = len(uniqueVersions)
	}
	goVersions := make([]string, numVersions)
	for i, v := range uniqueVersions[:numVersions] {
		goVersions[i] = v.String()
	}

	return goVersions
}

func uniqueVersionSlice(versions []*version.Version) []*version.Version {
	keys := make(map[string]bool)
	unique := []*version.Version{}
	for _, v := range versions {
		if _, ok := keys[v.String()]; !ok {
			keys[v.String()] = true
			unique = append(unique, v)
		}
	}
	return unique
}
