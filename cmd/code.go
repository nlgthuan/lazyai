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

// codeCmd represents the code command
var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Generate prompt to ask LLM to implement new features",
	Long: `Generate prompt for code. For example:

git ls-files | tail -n +1 | lazyai code`,
	Run: generateCode,
}

func init() {
	rootCmd.AddCommand(codeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// codeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// codeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func generateCode(cmd *cobra.Command, args []string) {
	scanner := bufio.NewScanner(os.Stdin)
	codeBlock := ""
	for scanner.Scan() {
		codeBlock += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	fmt.Println(`Given the below code, requirement and general rule:

<code>
` + codeBlock + `</code>

<requirement>
Help me <do stuff, add more explanation and constraint>.
</requirement>

<general_rule>
As a lead developer of the project, you will start with high level ideas on how you would implement this, considering all security and performance issues with it.
Think for as much as you want. Then, for now and the rest of this session:

- Only output code AFTER I have reviewed the strategy. You may realize that you need some file to know more about the system implementation, so feel free to ask me for it.
- Make sure you output professional-grade, production-ready code that is clean, optimized, maintainable and follow best practices.
- Print the code addition or change in 'diff' format (meaning they have the filename, the lines that got changed, and ignore the parts that is unchanged).
- The change must be compatible to be used in the 'patch' program to apply the changes. Follow the line numbers correctly, especially the starting line of the change.
- If printing out 'diff' file, then do not print out any other code block.
</general_rule>`)
}
