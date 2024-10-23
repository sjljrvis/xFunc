package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type CodeblockFile struct {
	Name   string
	Number int
}

func GenerateCommands(dir string) []string {
	cmds := []string{}
	dirPath := dir

	// Read the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	// Compile the regex pattern
	pattern := regexp.MustCompile(`^codeblock_(\d+)\.sh$`)

	var codeblockFiles []CodeblockFile
	for _, file := range files {
		if !file.IsDir() {
			matches := pattern.FindStringSubmatch(file.Name())
			if matches != nil {
				number, _ := strconv.Atoi(matches[1])
				codeblockFiles = append(codeblockFiles, CodeblockFile{
					Name:   file.Name(),
					Number: number,
				})
			}
		}
	}

	// Sort the matching files based on the number
	sort.Slice(codeblockFiles, func(i, j int) bool {
		return codeblockFiles[i].Number < codeblockFiles[j].Number
	})

	for _, file := range codeblockFiles {
		cmd := fmt.Sprintf("sh %s", file.Name)
		cmds = append(cmds, cmd)
	}
	return cmds
}

func SplitIntoCodeBlocksAndSave(input string, outputDir string) error {
	lines := strings.Split(input, "\n")
	var currentBlock strings.Builder
	inCodeBlock := false
	language := ""
	blockCount := 0

	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "```") {
			if inCodeBlock {
				// End of a code block
				blockCount++
				err := saveCodeBlock(outputDir, language, currentBlock.String(), blockCount)
				if err != nil {
					return err
				}
				currentBlock.Reset()
				inCodeBlock = false
				language = ""
			} else {
				// Start of a code block
				inCodeBlock = true
				language = strings.TrimPrefix(trimmedLine, "```")
			}
		} else if inCodeBlock {
			currentBlock.WriteString(line + "\n")
		}
	}

	// Handle case where the last code block isn't closed
	if inCodeBlock {
		blockCount++
		err := saveCodeBlock(outputDir, language, currentBlock.String(), blockCount)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractFileName(content string) string {
	filename := ""
	pattern := `(?m)^#\s*filename:\s*(\S+)`

	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		filename = match[1]
	}
	return filename
}

func saveCodeBlock(outputDir, language, content string, blockCount int) error {
	extension := getExtensionForLanguage(language)
	filename := extractFileName(content)
	if len(filename) == 0 {
		filename = fmt.Sprintf("codeblock_%d%s", blockCount, extension)
	}
	filepath := filepath.Join(outputDir, filename)

	err := ioutil.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %v", filepath, err)
	}

	log.Printf("[EXECUTOR] : Created file: %s\n", filepath)
	return nil
}

func getExtensionForLanguage(language string) string {
	switch strings.ToLower(language) {
	case "python":
		return ".py"
	case "javascript":
		return ".js"
	case "java":
		return ".java"
	case "go":
		return ".go"
	case "ruby":
		return ".rb"
	case "c++", "cpp":
		return ".cpp"
	case "c":
		return ".c"
	case "bash", "shell":
		return ".sh"
	default:
		return ".sh"
	}
}
