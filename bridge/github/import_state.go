package github

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

const defaultFilePath = "./git-bug-import"
const commentSymbol = "#"

type issueState int

const (
	new issueState = iota
	imported
	importErr
)

type importedIssue struct {
	issueNumber githubv4.Int
	importTime  time.Time
}

type importIssueErr struct {
	issueNumber githubv4.Int
	errTime     time.Time
}

type importState struct {
	filePath string

	fileLines         []string
	fileLinesToIssues map[int]githubv4.Int

	issues []githubv4.Int

	issueStates map[githubv4.Int]issueState
	importTime  map[githubv4.Int]time.Time

	isLoadedFromFile bool
}

func newImportState() importState {
	return newImportStateFromFile(defaultFilePath)
}

func newImportStateFromFile(filePath string) importState {
	return importState{
		filePath: filePath,

		fileLines:         make([]string, 0),
		fileLinesToIssues: make(map[int]githubv4.Int),

		issues: make([]githubv4.Int, 0),

		issueStates: make(map[githubv4.Int]issueState),
		importTime:  make(map[githubv4.Int]time.Time),

		isLoadedFromFile: false,
	}
}

func (s *importState) tryLoadFromFile() {
	file, err := os.Open(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		// TODO(as)
		fmt.Println("ERROR: opening file ", s.filePath)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		s.parse(line)
	}
	s.isLoadedFromFile = true
}

func (s *importState) parse(line string) {
	lineIdx := len(s.fileLines)
	s.fileLines = append(s.fileLines, line)
	commentIdx := strings.Index(line, commentSymbol)
	trimmedLine := line
	if commentIdx >= 0 {
		trimmedLine = line[:commentIdx]
	}
	trimmedLine = strings.TrimSpace(trimmedLine)
	if len(trimmedLine) <= 0 {
		return
	}
	issueNumInt, err := strconv.Atoi(trimmedLine)
	if err != nil {
		// TODO(as)
		fmt.Println("ERROR: ", err)
	} else {
		issueNumber := githubv4.Int(int32(issueNumInt))
		s.issues = append(s.issues, issueNumber)
		s.issueStates[issueNumber] = new
		s.fileLinesToIssues[lineIdx] = issueNumber
	}
}

func (s *importState) isLoaded() bool {
	return s.isLoadedFromFile
}

// TODO(as) rename "open" to xxx ?
func (s *importState) addNewIssues(issues []githubv4.Int) {
	s.issues = append(s.issues, issues...)
	for _, issue := range issues {
		s.issueStates[issue] = new
	}
}

func (s *importState) issuesToImport() []githubv4.Int {
	result := []githubv4.Int{}
	for _, issue := range s.issues {
		if s.issueStates[issue] == new {
			result = append(result, issue)
		}
	}
	return result
}

func (s *importState) setImportSuccess(issue githubv4.Int) {
	s.issueStates[issue] = imported
	s.importTime[issue] = time.Now()
	s.writeToFileSystem()
}

func (s *importState) setImportError(issue githubv4.Int) {
	s.issueStates[issue] = importErr
	s.importTime[issue] = time.Now()
	s.writeToFileSystem()
}

func writeNewIssue(file *os.File, issue githubv4.Int) error {
	_, err := file.WriteString(to_str(issue) + "\n")
	return err
}

func writeImportedIssue(file *os.File, issue githubv4.Int, importTime time.Time) error {
	str := fmt.Sprintf("%s %s %s imported %s %s\n", commentSymbol, to_str(issue), commentSymbol, commentSymbol, importTime.String())
	_, err := file.WriteString(str)
	return err
}

func writeImportErrIssue(file *os.File, issue githubv4.Int, importTime time.Time) error {
	str := fmt.Sprintf("%s %s import error %s %s\n", to_str(issue), commentSymbol, commentSymbol, importTime.String())
	_, err := file.WriteString(str)
	return err
}

func (s *importState) writeToFileSystem() {
	file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		// TODO(as)
		fmt.Println("(A) error writing file")
		return
	}
	defer file.Close()

	var done interface{}
	writtenToFile := map[githubv4.Int]interface{}{}

	for idx, line := range s.fileLines {
		var _ int
		var err error
		if issue, ok := s.fileLinesToIssues[idx]; ok {
			// this line corresponds to an issue
			switch s.issueStates[issue] {
			case new:
				writeNewIssue(file, issue)
			case imported:
				writeImportedIssue(file, issue, s.importTime[issue])
			case importErr:
				writeImportErrIssue(file, issue, s.importTime[issue])
			}
			writtenToFile[issue] = done
		} else {
			// this line is a comment
			_, err = file.WriteString(line + "\n")
		}
		if err != nil {
			fmt.Println("(A) error writing file")
		}
	}
	for _, issue := range s.issues {
		if _, ok := writtenToFile[issue]; ok {
			continue
		}
		switch s.issueStates[issue] {
		case new:
			writeNewIssue(file, issue)
		case imported:
			writeImportedIssue(file, issue, s.importTime[issue])
		case importErr:
			writeImportErrIssue(file, issue, s.importTime[issue])
		}
		writtenToFile[issue] = done
	}
}

func to_str(issueNumber githubv4.Int) string {
	return strconv.Itoa(int(issueNumber))
}
