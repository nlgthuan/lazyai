package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// prCmd represents the pr command
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Generate a prompt for Pull Request description based on git diff",
	Long: `The 'pr' command generates a detailed Pull Request description by analyzing the differences between your current branch and a specified base branch.

Configuration:
Ensure your git repository has a main or master branch, or specify a different base branch using the --base flag.

This command is designed to be used in conjunction with other commands that can process the generated description. For example:
    lazyai pr | lazyai sdchat`,
	Run: runPrCmd,
}

var baseBranch string

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.Flags().StringVarP(&baseBranch, "base", "b", "", "Base branch for git diff")
}

func runPrCmd(cmd *cobra.Command, args []string) {
	if baseBranch == "" {
		var err error
		baseBranch, err = getDefaultBranch()
		if err != nil {
			log.Fatalf("Failed to determine default branch: %v\n", err)
		}
	}

	diff, err := getGitDiff(baseBranch)
	if err != nil {
		log.Fatalf("Failed to get git diff: %v\n", err)
	}

	printPRDescription(diff)
}

func getDefaultBranch() (string, error) {
	gbCmd := exec.Command("git", "branch")

	output, err := gbCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run git branch: %v", err)
	}

	branches := strings.Split(string(output), "\n")

	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "main" {
			return "main", nil
		} else if branch == "master" {
			return "master", nil
		}
	}

	return "", fmt.Errorf("no main or master branch found")
}

func getGitDiff(baseBranch string) (string, error) {
	diffCmd := exec.Command("git", "diff", baseBranch)

	diff, err := diffCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run git diff: %v", err)
	}

	return string(diff), nil
}

func printPRDescription(diff string) {
	fmt.Println(`Help me generate PR description for the below git diff:
<diff>
` + diff + `</diff>

Note that the format of the PR should follow this one:` + "```" + `
## Description
This Pull Request introduces several key functionalities aimed at enhancing the user authentication and verification process in the SkyDeck Control Center (CC). Specifically, it allows users to sign up using email and password, restricts access until SMS verification is completed, and sets up a webhook to handle inbound SMS verification messages from Twilio. Additionally, it provides users with the ability to confirm the submission of their verification SMS.

## Summary of Changes
1. **Email and Password Signup for Control Center**:
   - Configured necessary settings in` + "`settings.py`" + `for handling signups.
   - Updated the signup template to include terms of use and privacy policy agreements.

2. **Restrict Access Until SMS Verification**:
   - Implemented middleware to redirect users to the verification instruction page if they are not verified.
   - Created views and templates for displaying SMS verification instructions.

3. **Additional Updates**:
   - Added tests for new models, views, and middleware to ensure robust functionality.
   - Updated styling and templates to improve user experience during the signup and verification process.

## Related Stories

- [429 - As a user, I can sign up for Control Center using email and password](https://www.pivotaltracker.com/story/show/187954429)
- [451 - Restrict CC access until SMS verification is completed](https://www.pivotaltracker.com/story/show/187975451)
` + "```")
}
