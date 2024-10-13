#!/bin/bash

# Function to ask for commit type using gum
ask_commit_type() {
    commit_type=$(gum choose "[f] - feature" "[c] - chore" "[b] - bug" --header "Story type?" | awk '{print substr($0, 1, 3)}')
    echo "$commit_type"
}

# Function to get git diff
get_git_diff() {
    git diff HEAD
}

# Function to generate a prompt for the user
generate_prompt() {
    local changes="$1"

    prompt="Please generate descriptive commit message for the following changes:\n\n$changes\n\n Just output the commit message, do not wrap it in anything.
    \n
    The first line of the commit message should be a concise name for the commit. \n
    Then in the body, we provide more context about the change in form of list, start with a dash. \n
    For example: \n

        This is a concise name of the commit \n\n
        - Add a new user model\n
        - Refactor views\n
    "
    echo "$prompt"
}

# Parse command line arguments
skip_story=false
while getopts "n" opt; do
    case $opt in
        n)
            skip_story=true
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            exit 1
            ;;
    esac
done

# Main script execution
changes=$(get_git_diff)

if [ -z "$changes" ]; then
    echo "No changes detected. Exiting."
    exit 0
else
    commit_type=$(ask_commit_type)

    if [ "$skip_story" = false ]; then
        story_url=$(lazyai pickPT -l)
        story_id=$(echo "$story_url" | awk '{print substr($0, length($0)-2)}')
    fi

    prompt=$(generate_prompt "$changes")
    echo "$prompt"
    ai_res=$(printf "%s" "$prompt" | xargs -0 -I {} lazyai sdchat "{}")

    if [ "$skip_story" = false ]; then
        commit_msg=$(echo "${commit_type} ${story_id} - ${ai_res}\n${story_url}")
    else
        commit_msg=$(echo "${commit_type} - ${ai_res}")
    fi

    echo "$commit_msg" | gum write --width 0 --height 50 --char-limit 0 | git commit -F -
fi
