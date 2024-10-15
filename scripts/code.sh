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
0. You are a senior developer who value best practices, and alwasy produce good, clean code.
1. Let's implement the task step by step. I will need to adjust your solution along the way.
2. Ensure the output is production-ready quality code that is clean, optimized, and maintainable.
3. The code should follow best practices and adhere to the project's coding standards.
EOF

${EDITOR:-vi} "$temp_file"

# Output the edited content
cat "$temp_file" | xargs -0 -I {} lazyai sdchat "{}" -n

# Remove the temporary file
rm "$temp_file"
