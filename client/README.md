# Python client for minimal install Git-based decentralized version control system

## Prerequisites

Follow instructions in the [project's README file](../README.md) to install required software, start the Hyperledger Fabric network and IPFS daemon.

## How to run

`python3 client.py <command_name> <arg1> <arg2> etc.`

## Supported commands

- **register**: Register as a new developer
  - Usage: `register <username> <email>`
    - Example: `register johndoe johndoe@email.com`
  - Will save the user's private key as a PEM file in the '.ssh/' directory
- **login**: Login as a developer
  - Usage: `login <username> <path_to_private_key>`, where `<path_to_private_key>` is the relative path to where the private key is located
  - Note: Only one developer may be logged in at a time. Log out first to switch developers.
- **regenerateKey**: Regenerate private key for currently connected user
  - Usage: `regenerateKey`
- **logout**: Logout currently connected developer
  - Usage: `logout`
- **init**: Initialize new repo at 'repo_parent_directory/repo_name' and push it to blockchain
  - Usage: `init <repo_name> <optional:repo_parent_directory>`
- **clone**: Clone repo from blockchain locally to 'repo_parent_directory/repo_name'
  - Usage: `clone <author> <repo_name> <optional:repo_parent_directory>`
  - Notes:
    - Developer must have read access to the repo to clone it
    - This command may also be used to update all branches of the repo
- **delete**: Delete repo from blockchain and from local directory
  - Usage: `delete <author> <repo_name> <optional:repo_parent_directory>`
  - Note: Developer must have owner access to delete the repo
- **addBranch**: Add a branch at currently checked out commit to repo
  - Usage: `addBranch <author> <repo_name> <branch_name> <optional:repo_parent_directory>`
  - Note: Developer must have read and write access to add a branch
- **deleteBranch**: Delete a branch from the repo
  - Usage: `deleteBranch <author> <repo_name> <branch_name> <optional:repo_parent_directory>`
  - Note: Developer must have read and write access to delete a branch
- **checkout**: Check out a branch from the repo
  - Usage: `checkout <author> <repo_name> <branch_name> <optional:repo_parent_directory>`
- **pull**: Pull commits from the blockchain for given branch from the repo
  - Usage: `pull <author> <repo_name> <branch_name> <optional:repo_parent_directory>`
  - Note: Developer must have read and write access to pull a branch
- **commit**: Add one commit on given branch and push it to the blockchain
  - Usage: `commit <author> <repo_name> <branch_name> <commit_message> <also_add_unstaged_files> <optional:repo_parent_directory>`
  - Notes:
    - Developer must have read and write access to commit and push to a branch
    - If 'also_add_unstaged_files' is set to 'true', unstaged files will also be added to the commit
    - Commit will be reverted if push is rejected by the blockchain
- **push**: Push existing commits on given branch to the blockchain
  - Usage: `push <author> <repo_name> <branch_name> <optional:repo_parent_directory>`
  - Notes:
    - Developer must have read and write access to push to a branch
    - If branch is not up to date with the blockchain, the push is canceled, prompting the user to pull again
    - Commits will be reverted if push is rejected by the blockchain
- **updateRepoAccess**: Update the access permissions for a user on a repo
  - Usage: `updateRepoAccess <repo_author> <repo_name> <username_to_authorize> <access_value>`
  - Notes:
    - Developer must have owner access to change access permissions for a repo
    - 'username_to_authorize' must be the user name of a registered user
    - 'access_value' must be one of 'ReadAccess', 'ReadWriteAccess' or 'OwnerAccess'
- **queryRepoAccess**: Get access permissions of all users of a repo
  - Usage: `queryRepoAccess <repo_author> <repo_name>`
