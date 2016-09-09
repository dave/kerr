package main

// ke: {"package": {"notest": true}}

import (
	"log"
	"path/filepath"

	"strings"

	"fmt"

	"flag"

	"os"

	"github.com/davelondon/gopackages"
	"github.com/davelondon/kerr"
	"github.com/davelondon/kerr/kcov/scanner"
	"github.com/davelondon/kerr/kcov/tester"
	"golang.org/x/tools/cover"
)

// Profile represents the profiling data for a specific file.
type Profile struct {
	FileName string
	Mode     string
	Blocks   []*ProfileBlock
	Exclude  bool
}

// ProfileBlock represents a single block of profiling data.
type ProfileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt, Count      int
	Exclude             bool
}

func main() {

	recursive := flag.Bool("r", false, "If -r is spefified, gotests will recursively test all subdirectories")
	js := flag.Bool("js", false, "If -js is spefified, gotests will run js tests")
	flag.Parse()

	var err error

	baseDir := ""
	pkg := ""
	if flag.Arg(0) == "" {
		baseDir = flag.Arg(0)
		pkg, err = gopackages.GetPackageFromDir(os.Getenv("GOPATH"), baseDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		pkg = flag.Arg(0)
		baseDir, err = gopackages.GetDirFromPackage(os.Environ(), os.Getenv("GOPATH"), flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
	}

	source, err := scanner.Get(baseDir)
	if err != nil {
		log.Fatal(err)
	}

	if js != nil && *js {
		if err := tester.Js(source.JsTestPackages); err != nil {
			log.Fatal(err)
		}
	}

	var coverProfiles []*cover.Profile
	if recursive == nil {
		coverProfiles, err = tester.GetSingle(baseDir, pkg)
	} else {
		coverProfiles, err = tester.Get(baseDir)
	}
	if err != nil {
		log.Fatal(err)
	}

	profiles := importProfiles(coverProfiles)

	if err := excludePackages(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludeFiles(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludeWraps(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludePanics(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludeBlocks(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludeFuncs(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := excludeSkips(profiles, source); err != nil {
		log.Fatal(err)
	}

	if err := checkComplete(profiles, source); err != nil {
		log.Fatal(err)
	}

	out := exportProfiles(profiles)

	if err := tester.Save(out, filepath.Join(baseDir, "coverage.out")); err != nil {
		log.Fatal(err)
	}
}

func importProfiles(coverProfiles []*cover.Profile) map[string]*Profile {
	profiles := map[string]*Profile{}
	for _, p := range coverProfiles {
		profile := &Profile{
			FileName: p.FileName,
			Mode:     p.Mode,
		}
		for _, b := range p.Blocks {
			block := &ProfileBlock{
				StartLine: b.StartLine,
				EndLine:   b.EndLine,
				StartCol:  b.StartCol,
				EndCol:    b.EndCol,
				NumStmt:   b.NumStmt,
				Count:     b.Count,
			}
			profile.Blocks = append(profile.Blocks, block)
		}
		profiles[p.FileName] = profile
	}
	return profiles
}

func exportProfiles(profiles map[string]*Profile) []*cover.Profile {
	var coverProfiles []*cover.Profile
	for _, p := range profiles {
		if p.Exclude {
			continue
		}
		profile := &cover.Profile{
			FileName: p.FileName,
			Mode:     p.Mode,
		}
		for _, b := range p.Blocks {
			if b.Exclude {
				continue
			}
			block := cover.ProfileBlock{
				StartLine: b.StartLine,
				EndLine:   b.EndLine,
				StartCol:  b.StartCol,
				EndCol:    b.EndCol,
				NumStmt:   b.NumStmt,
				Count:     b.Count,
			}
			profile.Blocks = append(profile.Blocks, block)
		}
		coverProfiles = append(coverProfiles, profile)
	}
	return coverProfiles
}

func checkComplete(profiles map[string]*Profile, source *scanner.Source) error {
	var errors []string
	for _, profile := range profiles {
		if profile.Exclude {
			continue
		}
		pkg := getPackage(profile)
		_, ok := source.CompletePackages[pkg]
		if ok {
			for _, block := range profile.Blocks {
				if block.Exclude {
					continue
				}
				if block.Count == 0 {
					errors = append(errors, fmt.Sprintf("Untested code in %s:%d-%d", profile.FileName, block.StartLine, block.EndLine))
				}
			}
		}
	}
	if len(errors) > 0 {
		fmt.Println(strings.Join(errors, "\n"))
		return kerr.New("GNLYPXHTNF", "Untested code in %d places", len(errors))
	}
	return nil
}

func excludePanics(profiles map[string]*Profile, source *scanner.Source) error {
	for _, def := range source.Panics {
		p, ok := profiles[def.File]
		if !ok {
			continue
		}
		for _, b := range p.Blocks {
			if b.StartLine <= def.Line && b.EndLine >= def.Line && b.Count == 0 {
				b.Exclude = true
				fmt.Printf("Excluding panic from %s:%d\n", def.File, def.Line)
			}
		}
	}
	return nil
}

func excludeWraps(profiles map[string]*Profile, source *scanner.Source) error {
	for _, def := range source.Wraps {
		p, ok := profiles[def.File]
		if !ok {
			continue
		}
		for _, b := range p.Blocks {
			if b.StartLine <= def.Line && b.EndLine >= def.Line && b.Count == 0 {
				b.Exclude = true
				fmt.Printf("Excluding Wrap %s from %s:%d\n", def.Id, def.File, def.Line)
			}
		}
	}
	return nil
}

func excludeBlocks(profiles map[string]*Profile, source *scanner.Source) error {
	for _, eb := range source.ExcludedBlocks {
		p, ok := profiles[eb.File]
		if !ok {
			continue
		}
		for _, b := range p.Blocks {
			actualLine := eb.Line + 1
			if b.StartLine <= actualLine && b.EndLine >= actualLine && b.Count == 0 {
				b.Exclude = true
				fmt.Printf("Excluding block from %s:%d\n", eb.File, eb.Line)
			}
		}
	}
	return nil
}

func excludeFuncs(profiles map[string]*Profile, source *scanner.Source) error {
	for _, ef := range source.ExcludedFuncs {
		p, ok := profiles[ef.File]
		if !ok {
			continue
		}
		for _, b := range p.Blocks {
			if b.StartLine <= ef.LineEnd && b.EndLine >= ef.LineStart && b.Count == 0 {
				b.Exclude = true
				fmt.Printf("Excluding func from %s:%d-%d\n", ef.File, ef.LineStart, ef.LineEnd)
			}
		}
	}
	return nil
}

func excludePackages(profiles map[string]*Profile, source *scanner.Source) error {
	for _, profile := range profiles {
		pkg := getPackage(profile)
		if _, ok := source.ExcludedPackages[pkg]; ok {
			profile.Exclude = true
			fmt.Printf("Excluding package %s - %s\n", pkg, profile.FileName)
		}
	}
	return nil
}

func excludeFiles(profiles map[string]*Profile, source *scanner.Source) error {
	for _, profile := range profiles {
		if _, ok := source.ExcludedFiles[profile.FileName]; ok {
			profile.Exclude = true
			fmt.Printf("Excluding file %s\n", profile.FileName)
		}
	}
	return nil
}

func excludeSkips(profiles map[string]*Profile, source *scanner.Source) error {
	for id, _ := range source.Skipped {
		def, ok := source.All[id]
		if ok {
			p, ok := profiles[def.File]
			if !ok {
				continue
			}
			for _, b := range p.Blocks {
				if b.StartLine <= def.Line && b.EndLine >= def.Line && b.Count == 0 {
					b.Exclude = true
					fmt.Printf("Excluding skipped error %s from %s:%d\n", id, def.File, def.Line)
				}
			}
		}
	}
	return nil
}

func getPackage(profile *Profile) string {
	dir, _ := filepath.Split(profile.FileName)
	if strings.HasSuffix(dir, "/") {
		dir = dir[:len(dir)-1]
	}
	// dir is relative to the gopath/src, so we can just convert to slashes to get the package path.
	return filepath.ToSlash(dir)
}
