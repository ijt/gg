{
    "cmd_aliases": [
        "pr"
    ],
    "cmd_class": "basic",
    "date": "2018-07-16 08:42:42-07:00",
    "lastmod": "2018-07-16 08:55:07-07:00",
    "title": "gg requestpull",
    "usage": "gg requestpull [-n] [BRANCH]"
}

create a GitHub pull request

<!--more-->

Create a new GitHub pull request for the given branch (defaults to the
one currently checked out). The source will be inferred from the
branch's remote push information and the destination will be inferred
from upstream fetch information. This command does not push any new
commits; it just creates a pull request.

For non-dry runs, you must create a [personal access token][] at
https://github.com/settings/tokens/new and save it to
`$XDG_CONFIG_HOME/gg/github_token` (or in any other directory
in `$XDG_CONFIG_DIRS`). By default, this would be
`~/.config/gg/github_token`. gg needs at least `public_repo` scope
to be able to create pull requests, but you can grant `repo` scope to
create pull requests in any repositories you have access to.

[personal access token]: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/

## Options

<dl class="flag_list">
	<dt>-n</dt>
	<dt>-dry-run</dt>
	<dd>prints the pull request instead of creating it</dd>
</dl>