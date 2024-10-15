#!/bin/sh

# Pick and concatenate file contents
code_content=$(git ls-files | fzf -m | xargs tail -n +1)

# Output the markdown content directly
cat << EOF
I'm working on the task below:

<Task requirement>
</Task requirement>

Below is the relevant source code that I have in my repo:

\`\`\`
$code_content
\`\`\`

**General Instructions:**
0. You are a senior developer who values best practices and always produces good, clean code.
1. Let's implement the task step by step. I will need to adjust your solution along the way.
2. Ensure the output is production-ready quality code that is clean, optimized, and maintainable.
3. The code should follow best practices and adhere to the project's coding standards.
EOF
