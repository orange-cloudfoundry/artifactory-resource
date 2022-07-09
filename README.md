# Artifactory Resource

A concourse resource for push and download files from/to artifactory with semver.

## Source Configuration

* `url`: *Required.* Artifactory base url

* `repository`: *Required.* Directory to watch, read and write files

* `filter`: *Optional.* Filter files in repository that match given regxp
  * use named groups to extract content from filename and customize version sorting:
    * `?P<version>`: sort as semver
    * `?P<asc>`:  sort alphabetically, asc
    * `?P<desc>`: sort alphabetically, desc
  * when property **is not set**:
    * all files in given repository path are considered
    * artifactory's last modified timestamp is used for versioning
  * when property **is set**:
    * only files matching filter regex are considered
    * given `<version>`, `asc` or `desc` named groups are used for versioning
    * artifactory's last modified timestamp is used when none of `version`, `asc` or `desc`
      are present in filter named groups

* `user`: *Optional.* Artifactory username.

* `password`: *Optional.* Artifactory password.

* `ssh_key`: *Optional.* Artifactory ssh key.

* `log_level`: *Default: `ERROR`* Set the verbosity of logs, other values are: `ERROR`, `WARN`, `DEBUG`.

* `ca_cert`: *Optional.* Pass a certificate to access to your artifactory.

* `props`: *Optional.* Set of props to filter in check command and always include for out command
  given with the following format:
  ```yaml
  prop1:
    - value1
    - value2
  prop2: [ "value" ]
  ```

* `threads`: *Default: `3`* Number of transfer threads for in and out commands.

## Behavior

### `check`: Check for new files.

Find all files in `repository` matching the `filter` ordered according to used named groups
`version` (semver), `asc` (alphabetically), `desc` (reverse alphabetically) or ordered by the
modified timestamp if no group is given.


### `in`: Download a file from Artifactory


#### Parameters

* `min_split`: *Default: 5120* The minimum size permitted for splitting. Files larger than the
  specified number will be split into equally sized `split_count` segments. Any files smaller than
  the specified number will be downloaded in a single thread. If set to -1, files are not split.

* `split_count`: *Default: 3* The number of segments into which each file should be split for
  download (provided the artifact is over --min-split in size). To download each file in a
  single thread, set to 0.

* `props_filename`: *Optional.* When given, download properties associated to file and write it
  to given filename. File is written as YAML with the same format as `source.props`.

### `out`: Upload a file to artifactory.

#### Parameters

* `directory`: *Required.* Upload files from given directory that match `source.filter`. When
  multiple files match, they are all uploaded and version and meta refers to last matching file.

* `props`: *Optional.* Additional properties to add to uploaded file merged with `source.props`.
  Properties take precedence over `source.props` on collisions and given with the same format
  as `source.props`

* `props_filename`: *Optional.* Load additional properties from given yaml file merged with
  `source.props` and `params.props`. Defined properties takes precedence on collisions and given
  with the same format as `source.props`.

## Example

``` yaml
resource_types:
- name: artifactory
  type: docker-image
  source:
    repository: orangeopensource/artifactory-resource

resources:
- name: artifactory-resource
  type: artifactory
  source:
    url: https://my.artifactory.com
    user: myuser
    password: mypassword
    repository: bosh_release/credhub/
    filter: "credhub-v(?P<version>)\\.tgz"

jobs:
- name: build-rootfs
  plan:
  -
    # will fetch last available version of /bosh_release/credhub/credhub-*.tgz
    # and will download properties in ./spec.yml
    get: artifactory-resource
    params:
      props_filename: spec.yml
  -
    # some task that create ./output/credhub-v8.9.10.tgz
    task: { ... }
  -
    # will upload ./output/credhub-v8.9.10.tgz to /bosh_release/credhub/credhub-v8.9.10.tgz
    # and will set property built_by=concourse
    put: artifactory-resource
    params:
      directory: ./output/
      props_filename: inject.yml
      props:
        built_by:
        - concourse

```
