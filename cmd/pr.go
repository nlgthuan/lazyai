/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/spf13/cobra"
)

// prCmd represents the pr command
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Generate Pull Request Description",
	Run: func(cmd *cobra.Command, args []string) {
		diffCmd := exec.Command("ls", "-l")
		// get the output from the command execution
		diff, err := diffCmd.CombinedOutput()
		if err != nil {
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}

		fmt.Println(`Help me generate PR description for the below git diff:
<diff>
` + string(diff) + `</diff>

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
	},
}

func init() {
	rootCmd.AddCommand(prCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// prCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// prCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
