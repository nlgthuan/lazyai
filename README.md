# LazyAI

LazyAI is a command-line interface (CLI) tool designed to streamline your development workflow by integrating with AI services and project management tools. It helps you effortlessly generate commit messages, pull request descriptions, and interact with AI models.

## Installation

To install LazyAI, you need to have Go installed on your machine. Once you have Go set up, you can install LazyAI by running the following command:

```sh
go install github.com/nlgtEA/lazyai@latest
```

Make sure your `~/go/bin` is added to your system's `$PATH` so you can run the `lazyai` command from anywhere.

## Configuration

LazyAI requires a configuration file to store API tokens and other necessary settings. Create a file named `.lazyai.yml` in your home directory with the following structure:

```yaml
skydeck:
  accessToken: <your_access_token>
  refreshToken: <your_refresh_token>
  convoID: 0

pivotalTracker:
  apiToken: <your_api_token>
  projectID: <your_project_id>
  owner: <your_account_owner_name>
```

Replace the placeholders with your actual tokens and IDs.

## Usage


### Send a Message to SkyDeck

To send a message to the SkyDeck AI service, use:

```sh
lazyai sdchat "Your message here"
```

OR pipe from stdin

```sh
echo "Hello world!" | lazyai sdchat
```

### Retrieve a Pivotal Tracker Story

To retrieve the description of your active Pivotal Tracker story, use:

```sh
lazyai pickPT
```

### Others
For more details on each command, you can use the `--help` flag:

```sh
lazyai <command> --help
```

## Utilities in the `scripts` Folder

The `scripts` folder contains a set of utility scripts designed to streamline common development tasks related to Git operations and prompt generation. Below is a brief description of each script:

### `spr`

The `spr` script automates the process of creating a GitHub pull request. It leverages a series of tools to generate and edit a pull request description before submitting it. Here’s a step-by-step breakdown:

- **Prompt Generation**: It starts by using the `prompt` script with the `pr` template to generate a pull request description.
- **AI Assistance**: The description is then refined using `lazyai sdchat -n`.
- **Editing**: The user can edit the refined description using `sponge` and `vipe`, which allow for in-terminal editing.
- **Pull Request Creation**: Finally, `xargs` is used to pass the edited description to the `gh pr create` command, which creates the pull request on GitHub with the provided body.

### `scommit`

The `scommit` script simplifies the process of creating a commit with a descriptive message. Here's how it works:

- **Prompt Generation**: It uses the `prompt` script with the `commit` template to generate a commit message based on the current changes in the repository.
- **AI Assistance**: The generated message is refined using `lazyai sdchat -n`.
- **Editing**: The user can further refine the commit message using `sponge` and `vipe`.
- **Commit Creation**: The final message is used by `git commit -F -` to create a new commit with the specified message.

### `prompt`

The `prompt` script is a utility for generating templates that can assist in creating commit messages, pull request descriptions, and other code-related documentation. It supports different templates based on the provided option:

- **code**: Generates a template for code-related task descriptions.
- **commit**: Creates a template for drafting commit messages based on Git diffs.
- **pr**: Produces a template for drafting pull request descriptions, also based on Git diffs.

#### Usage

To use the `prompt` script, specify the template you want by using the `--pattern` or `-p` flag followed by the template name (`code`, `commit`, or `pr`). For example:

```bash
./prompt --pattern pr
```

This will generate a pull request description template.

## Contributing

Contributions are welcome! Feel free to submit a pull request or report any issues you encounter.

## License

This project is licensed under the MIT License.
