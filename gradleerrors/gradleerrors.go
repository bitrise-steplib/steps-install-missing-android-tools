package gradleerrors

import (
	"bufio"
	"fmt"
	"log"
	"regexp"
	"strings"
)

/*
MultipleFailuresFinder

FAILURE: Build completed with 2 failures.

1: Task failed with an exception.
-----------
* Where:
Build file '/bitrise/src/app/build.gradle' line: 14

* What went wrong:
A problem occurred evaluating project ':app'.
> /bitrise/src/apikey.properties (No such file or directory)

* Try:
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Exception is:
org.gradle.api.GradleScriptException: A problem occurred evaluating project ':app'
...

==============================================================================

2: Task failed with an exception.
-----------
* What went wrong:
A problem occurred configuring project ':app'.
> compileSdkVersion is not specified. Please add it to build.gradle

* Try:
> Run with --info or --debug option to get more log output.
> Run with --scan to get full insights.

* Exception is:
org.gradle.api.ProjectConfigurationException: A problem occurred configuring project ':app'.
*/
type MultipleFailuresFinder struct{}

func (f MultipleFailuresFinder) findErrors(out string) ([]string, error) {
	var relevantLines []string
	errorTypeDetected := false
	failureDetected := false

	var finder *whereWhatWentWrongFinder

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		if !errorTypeDetected {
			if regexp.MustCompile(`FAILURE: Build completed with (\d+) failures.`).FindString(line) != "" {
				errorTypeDetected = true
				relevantLines = append(relevantLines, line)
			}
		} else {
			if !failureDetected {
				if regexp.MustCompile(`(\d+): Task failed with an exception.`).FindString(line) != "" {
					if finder != nil {
						if !finder.whatWentWrongSectionFinished {
							return nil, fmt.Errorf("unexpected error structure: no what went wrong section found")
						}

						relevantLines = append(relevantLines, finder.relevantLines...)
					}

					failureDetected = true
					relevantLines = append(relevantLines, line)

					finder = &whereWhatWentWrongFinder{}
				}
			} else {
				if line == "==============================================================================" {
					failureDetected = false
					continue
				}

				if err := finder.find(line); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if finder != nil {
		if !finder.whatWentWrongSectionFinished {
			return nil, fmt.Errorf("unexpected error structure: no what went wrong section found")
		}

		relevantLines = append(relevantLines, finder.relevantLines...)
	}

	if !errorTypeDetected {
		return nil, nil
	}

	return []string{strings.Join(relevantLines, "\n")}, nil
}

/*
FailureFinder

FAILURE:

* Where: (optional)

* What went wrong:

* Try:

* Exception is:
*/
type whereWhatWentWrongFinder struct {
	whereSectionStarted          bool
	whereSectionFinished         bool
	whatWentWrongSectionStarted  bool
	whatWentWrongSectionFinished bool

	relevantLines []string
}

func (f *whereWhatWentWrongFinder) find(line string) error {
	if !f.whereSectionStarted {
		if strings.HasPrefix(line, "* Where:") {
			if f.whereSectionFinished {
				return fmt.Errorf("unexpected error structure: multiple where section")
			}

			f.whereSectionStarted = true
			f.relevantLines = append(f.relevantLines, line)
		}
	} else {
		if strings.TrimSpace(line) == "" {
			f.whereSectionStarted = false
			f.whereSectionFinished = true
		} else {
			f.relevantLines = append(f.relevantLines, line)
		}
	}

	if !f.whatWentWrongSectionStarted {
		if strings.HasPrefix(line, "* What went wrong:") {
			if f.whatWentWrongSectionFinished {
				return fmt.Errorf("unexpected error structure: multiple what went wrong section")
			}

			f.whatWentWrongSectionStarted = true
			f.relevantLines = append(f.relevantLines, line)
		}
	} else {
		if strings.TrimSpace(line) == "" {
			f.whatWentWrongSectionStarted = false
			f.whatWentWrongSectionFinished = true
		} else {
			f.relevantLines = append(f.relevantLines, line)
		}
	}

	return nil
}

type FailureFinder struct{}

func (f FailureFinder) findErrors(out string) ([]string, error) {
	var relevantLines []string
	errorTypeDetected := false

	finder := &whereWhatWentWrongFinder{}

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		if !errorTypeDetected {
			if strings.HasPrefix(line, "FAILURE: ") {
				errorTypeDetected = true
				relevantLines = append(relevantLines, line)
			}
		} else {
			if err := finder.find(line); err != nil {
				return nil, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if !errorTypeDetected {
		return nil, nil
	}

	if !finder.whatWentWrongSectionFinished {
		return nil, fmt.Errorf("unexpected error structure: no what went wrong section found")
	}

	relevantLines = append(relevantLines, finder.relevantLines...)

	return []string{strings.Join(relevantLines, "\n")}, nil
}

/*
ErrorCausedByFinder

Error: Could not find or load main class org.gradle.wrapper.GradleWrapperMain
Caused by: java.lang.ClassNotFoundException: org.gradle.wrapper.GradleWrapperMain
*/
type ErrorCausedByFinder struct{}

func (f ErrorCausedByFinder) findErrors(out string) ([]string, error) {
	var errorTypeDetected bool
	var relevantLines []string

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if !errorTypeDetected {
			if strings.HasPrefix(line, "Error: ") {
				errorTypeDetected = true
				relevantLines = append(relevantLines, line)
			}
		} else {
			if strings.HasPrefix(line, "Caused by: ") {
				relevantLines = append(relevantLines, line)
				break
			}
			return nil, fmt.Errorf("unexpected error structure")
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if len(relevantLines) == 0 {
		return nil, nil
	}

	return []string{strings.Join(relevantLines, "\n")}, nil
}

func findGradleErrors(out string) []string {
	return nil
}
