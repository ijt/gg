#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o pipefail

if [[ $# -gt 1 ]]; then
  echo "usage: misc/release.bash [VERSION]" 1>&2
  exit 64
fi
srcroot="$(dirname "$(dirname "${BASH_SOURCE[0]}")")" || exit 1
release_version="${1:-$(echo "$TRAVIS_TAG" | sed -n -e 's/v\([0-9].*\)/\1/p')}"
if [[ -z "$release_version" ]]; then
  echo "misc/release.bash: cannot infer version, please pass explicitly" 1>&2
  exit 1
fi
release_os="$(vgo env GOOS)" || exit 1
release_arch="$(vgo env GOARCH)" || exit 1
release_name="gg-${release_version}-${release_os}_${release_arch}"

echo "Creating ${release_name}.tar.gz..." 1>&2
stagedir="$(mktemp -d 2>/dev/null || mktemp -d -t gg_release)" || exit 1
trap 'rm -rf $stagedir' EXIT
distroot="$stagedir/$release_name"
mkdir "$distroot" || exit 1
cp "$srcroot/README.md" "$srcroot/CHANGELOG.md" "$srcroot/LICENSE" "$distroot/" || exit 1
"$srcroot/misc/build.bash" "$distroot/gg" "$release_version" || exit 1
tar -zcf - -C "$stagedir" "$release_name" > "${release_name}.tar.gz" || exit 1
