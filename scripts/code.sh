#!/bin/sh

# pick, concat file
code_content=$(git ls-files | fzf -m | xargs tail -n +1)

temp_file=$(mktemp /tmp/tempfile.XXXXXX)
mv "$temp_file" "$temp_file.md"
temp_file="$temp_file.md"

cat << EOF > "$temp_file"
I'm working on the task below:

<Task requirement>
</Task requirement>

Below is the relavant source code that I have in my repo:

\`\`\`
$code_content
\`\`\`

**General Instructions:**
1. Let's implement the task step by step. I will need to adjust your solution along the way.
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
- Please understand that: Incorrect context lines or misuse of symbols (e.g., using "+" for context lines) will prevent us from applying your Git diff output.
EOF


${EDITOR:-vi} "$temp_file"

edited_content=$(<"$temp_file")

echo "$edited_content"

rm "$temp_file"
