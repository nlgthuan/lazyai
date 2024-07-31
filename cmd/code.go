/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	codeFile    string
	requirement string
)

// codeCmd represents the code command
var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Generate prompt to ask LLM to implement new features",
	Long: `Generate prompt for code. For example:

git ls-files | tail -n +1 | lazyai code`,
	Run: generateCode,
}

func init() {
	codeCmd.Flags().StringVarP(&codeFile, "code", "c", "", "File containing the code to read")
	codeCmd.Flags().StringVarP(&requirement, "requirement", "r", "", "User story requirement")
	rootCmd.AddCommand(codeCmd)
}

func generateCode(cmd *cobra.Command, args []string) {
	var scanner *bufio.Scanner

	if codeFile != "" {
		file, err := os.Open(codeFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error opening file:", err)
			os.Exit(1)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	codeBlock := ""
	for scanner.Scan() {
		codeBlock += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Reading input:", err)
	}

	fmt.Println(`I'm working on this Pivotal Tracker story:

<User Story>
` + requirement + `
</User Story>

I will provide you my current source code. Please help me implement this story.

<Current Code>` + codeBlock +
		`</Current Code>

**General Instructions:**
1. Let's implement the story step by step. I will need to adjust your solution along the way.
2. Ensure the output is production-ready quality code that is clean, optimized, and maintainable.
3. The code should follow best practices and adhere to the project's coding standards.
4. Please present each code change in two formats:
  - Normal text format, with surrounding context so that I know where to place the code.
  - Standard unified Git diff format so that I can apply the code change using Git commands.
**Guidelines for Git Diff Output:**
- Do not include any comments or explanatory text within the diff output.
- Comment lines such as " // ... (rest of the file remains the same)" or "// ... (existing code)" in the diff output are unacceptable.
- Ensure that context lines are 100% accurate, including spaces, empty lines, and brackets.
- Even minor discrepancies in context lines can prevent the diff from being applied successfully.
- Preserve exact indentation for all added, removed, and context lines as they appear in the file.
- Use "+" for added lines, "-" for removed lines, and a space for context lines.
- Please understand that: Incorrect context lines or misuse of symbols (e.g., using "+" for context lines) will prevent us from applying your Git diff output.`)
}
