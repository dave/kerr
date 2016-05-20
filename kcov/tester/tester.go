package tester

// ke: {"package": {"notest": true}}

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davelondon/gopackages"
	"github.com/davelondon/kerr"
	"golang.org/x/tools/cover"
)

var baseDir string
var coverDir string
var merged []*cover.Profile

func Save(profiles []*cover.Profile, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return kerr.Wrap("TWQHWMXWMN", err)
	}
	defer f.Close()
	dumpProfiles(profiles, f)
	return nil
}

func Js(packages map[string]bool) error {
	for pkg, _ := range packages {
		fmt.Println("gopherjs test -v", pkg)
		exe := exec.Command("gopherjs", "test", "-v", pkg)
		exe.Stderr = os.Stderr
		exe.Stdout = os.Stdout
		if err := exe.Run(); err != nil {
			return kerr.Wrap("DMLLRIOUFL", err)
		}
	}
	return nil
}

func GetSingle(base string, singlePackage string) ([]*cover.Profile, error) {
	return get(base, singlePackage)
}
func Get(base string) ([]*cover.Profile, error) {
	return get(base, "")
}
func get(base string, singlePackage string) ([]*cover.Profile, error) {
	baseDir = base
	coverDir = filepath.Join(baseDir, ".coverage")

	testing := false
	if !testing {
		os.RemoveAll(coverDir)
		defer os.RemoveAll(coverDir)
	}
	if _, err := os.Stat(coverDir); os.IsNotExist(err) {
		os.Mkdir(coverDir, 0777)
	}
	singlePackageDir := ""
	if singlePackage != "" {
		var err error
		singlePackageDir, err = gopackages.GetDirFromPackage(os.Environ(), os.Getenv("GOPATH"), singlePackage)
		if err != nil {
			return nil, err
		}
	}

	// walk each file in the working directory
	walker := func(filename string, file os.FileInfo, err error) error {
		if strings.HasPrefix(getParentDir(filename), ".") {
			return filepath.SkipDir
		}
		if strings.HasPrefix(file.Name(), ".") {
			return nil
		}
		if file.IsDir() && (singlePackageDir == "" || singlePackageDir == filename) {
			return processDir(filename)
		}
		return nil
	}
	if err := filepath.Walk(baseDir, walker); err != nil {
		return nil, err
	}
	return merged, nil
}

func processCoverageFile(filename string) error {
	profiles, err := cover.ParseProfiles(filename)
	if err != nil {
		return kerr.Wrap("DVBQJEVTJU", err)
	}
	for _, p := range profiles {
		merged = addProfile(merged, p)
	}
	return nil
}

func processDir(dir string) error {

	files, _ := ioutil.ReadDir(dir)
	foundTest := false
	for _, f := range files {
		if strings.HasSuffix(f.Name(), "_test.go") {
			foundTest = true
		}
	}
	if !foundTest {
		return nil
	}

	relDir, err := filepath.Rel(filepath.Join(baseDir, ".."), dir)
	if err != nil {
		return kerr.Wrap("YYJUSYYGKE", err)
	}
	coverageFilename := strings.Replace(relDir, string(os.PathSeparator), "_", -1) + ".out"
	coverageFilePath := filepath.Join(coverDir, coverageFilename)

	if _, err := os.Stat(coverageFilePath); err == nil {
		return processCoverageFile(coverageFilePath)
	}

	os.Chdir(dir)
	outParam := fmt.Sprintf("-coverprofile=%s", coverageFilePath)
	exe := exec.Command("go", "test", outParam)
	b, err := exe.CombinedOutput()
	if strings.Contains(string(b), "no buildable Go source files in") {
		return nil
	}
	fmt.Println(string(b))
	if err != nil {
		return kerr.Wrap("GIDQRLQARA", err)
	}
	return processCoverageFile(coverageFilePath)
}

func getParentDir(filename string) string {
	dir, _ := filepath.Split(filename)
	dirs := strings.Split(dir, string(os.PathSeparator))
	return dirs[len(dirs)-2]
}

func mergeProfiles(p *cover.Profile, merge *cover.Profile) {
	if p.Mode != merge.Mode {
		log.Fatalf("cannot merge profiles with different modes")
	}
	// Since the blocks are sorted, we can keep track of where the last block
	// was inserted and only look at the blocks after that as targets for merge
	startIndex := 0
	for _, b := range merge.Blocks {
		startIndex = mergeProfileBlock(p, b, startIndex)
	}
}

func mergeProfileBlock(p *cover.Profile, pb cover.ProfileBlock, startIndex int) int {
	sortFunc := func(i int) bool {
		pi := p.Blocks[i+startIndex]
		return pi.StartLine >= pb.StartLine && (pi.StartLine != pb.StartLine || pi.StartCol >= pb.StartCol)
	}

	i := 0
	if sortFunc(i) != true {
		i = sort.Search(len(p.Blocks)-startIndex, sortFunc)
	}
	i += startIndex
	if i < len(p.Blocks) && p.Blocks[i].StartLine == pb.StartLine && p.Blocks[i].StartCol == pb.StartCol {
		if p.Blocks[i].EndLine != pb.EndLine || p.Blocks[i].EndCol != pb.EndCol {
			log.Fatalf("OVERLAP MERGE: %v %v %v", p.FileName, p.Blocks[i], pb)
		}
		p.Blocks[i].Count |= pb.Count
	} else {
		if i > 0 {
			pa := p.Blocks[i-1]
			if pa.EndLine >= pb.EndLine && (pa.EndLine != pb.EndLine || pa.EndCol > pb.EndCol) {
				log.Fatalf("OVERLAP BEFORE: %v %v %v", p.FileName, pa, pb)
			}
		}
		if i < len(p.Blocks)-1 {
			pa := p.Blocks[i+1]
			if pa.StartLine <= pb.StartLine && (pa.StartLine != pb.StartLine || pa.StartCol < pb.StartCol) {
				log.Fatalf("OVERLAP AFTER: %v %v %v", p.FileName, pa, pb)
			}
		}
		p.Blocks = append(p.Blocks, cover.ProfileBlock{})
		copy(p.Blocks[i+1:], p.Blocks[i:])
		p.Blocks[i] = pb
	}
	return i + 1
}

func addProfile(profiles []*cover.Profile, p *cover.Profile) []*cover.Profile {
	i := sort.Search(len(profiles), func(i int) bool { return profiles[i].FileName >= p.FileName })
	if i < len(profiles) && profiles[i].FileName == p.FileName {
		mergeProfiles(profiles[i], p)
	} else {
		profiles = append(profiles, nil)
		copy(profiles[i+1:], profiles[i:])
		profiles[i] = p
	}
	return profiles
}

func dumpProfiles(profiles []*cover.Profile, out io.Writer) {
	if len(profiles) == 0 {
		return
	}
	fmt.Fprintf(out, "mode: %s\n", profiles[0].Mode)
	for _, p := range profiles {
		for _, b := range p.Blocks {
			fmt.Fprintf(out, "%s:%d.%d,%d.%d %d %d\n", p.FileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol, b.NumStmt, b.Count)
		}
	}
}
