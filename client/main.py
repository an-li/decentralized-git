#!/usr/bin/env python3

import json
import os
import shutil
import sys
from datetime import datetime

import pytz
from hfc.fabric.chaincode import ChaincodeExecutionError

from cryptography_utils.rsa_key_generator import (
    extract_public_key_bytes,
    generate_rsa_key_pair,
    read_rsa_private_key_from_pem,
)
from data.models import AccessLog, Branch, Commit, Repository, UserAccess
from git_client.client import (
    add_pulled_commits_to_branch,
    checkout_branch,
    commit,
    commits_after_specified_to_chaincode_structure,
    convert_repository_to_chaincode_structure,
    create_new_branch,
    delete_branch,
    get_all_commit_hashes_for_branch_from_chaincode,
    get_all_commit_hashes_for_local_branch,
    get_latest_common_commit_hash,
    get_latest_commit_hash,
    initialize_repo,
    initialize_repo_from_chaincode_structure,
    revert_commit,
    set_git_user_config,
)
from hyperledger.client import invoke_function


def get_arg_at_position(
    args_list: list[str], index: int, default_value=None
) -> str | None:
    try:
        return args_list[index]
    except IndexError:
        return default_value


if __name__ == "__main__":
    command = sys.argv[1]
    other_args = sys.argv[2:]

    match command:
        case "register":
            name = other_args[0]
            email = other_args[1]
            private_key_path = f".ssh/{name}_{datetime.now().timestamp()}.pem"

            private_key, public_key = generate_rsa_key_pair(private_key_path)
            response = invoke_function("registerNewUser", [name, email, public_key])
            print(f"Your private key is located at {private_key_path}")
        case "login":
            # Args: name, private_key_path
            name = other_args[0]
            rsa_private_key_path = other_args[1]

            private_key = read_rsa_private_key_from_pem(rsa_private_key_path)
            public_key = extract_public_key_bytes(private_key.public_key()).decode(
                "utf-8"
            )

            response = invoke_function("logIn", [name, public_key])
            logged_in_user = json.loads(
                invoke_function(
                    "whoAmI",
                    [],
                )
            )
            set_git_user_config(logged_in_user["name"], logged_in_user["email"])
        case "regenerateKey":
            # Args: private_key_path
            logged_in_user = json.loads(
                invoke_function(
                    "whoAmI",
                    [],
                )
            )
            private_key_path = (
                f".ssh/{logged_in_user['name']}_{datetime.now().timestamp()}.pem"
            )

            private_key, public_key = generate_rsa_key_pair(private_key_path)
            response = invoke_function("changePublicKey", [public_key])
            print(f"Your new private key is located at {private_key_path}")
        case "logout":
            # No args required
            response = invoke_function("logOut", other_args)
        case "init":
            repo_name = other_args[0]
            repo_parent_directory = get_arg_at_position(other_args, 1, os.getcwd())

            logged_in_user = json.loads(
                invoke_function(
                    "whoAmI",
                    [],
                )
            )

            author = logged_in_user["name"]

            initialize_repo(
                f"{repo_parent_directory}/{repo_name}", author, logged_in_user["email"]
            )
            repository_from_blockchain = convert_repository_to_chaincode_structure(
                author, repo_name, repo_parent_directory
            )
            repository_from_blockchain.accessLogs = [
                AccessLog(
                    authorizer=author,
                    authorized=author,
                    timestamp=datetime.now().astimezone(pytz.utc),
                    userAccess=UserAccess.OwnerAccess,
                )
            ]
            repository_from_blockchain.access = {author: UserAccess.OwnerAccess}

            response = invoke_function(
                "addNewRepo",
                [repository_from_blockchain.json()],
            )
        case "clone":
            author = other_args[0]
            repo_name = other_args[1]
            repo_parent_directory = get_arg_at_position(other_args, 2, os.getcwd())

            logged_in_user = json.loads(
                invoke_function(
                    "whoAmI",
                    [],
                )
            )

            response = invoke_function("clone", other_args[0:2])
            repository_from_blockchain = Repository.model_validate(json.loads(response))
            initialize_repo_from_chaincode_structure(
                repository_from_blockchain,
                repo_parent_directory=repo_parent_directory,
            )
        case "delete":
            author = other_args[0]
            repo_name = other_args[1]
            repo_parent_directory = get_arg_at_position(other_args, 2, os.getcwd())

            response = invoke_function("deleteRepo", other_args[0:2])
            shutil.rmtree(f"{repo_parent_directory}/{repo_name}", ignore_errors=True)
        case "addBranch":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            repo_parent_directory = get_arg_at_position(other_args, 3, os.getcwd())

            branch_chaincode = create_new_branch(
                repo_name, branch_name, repo_parent_directory
            )

            try:
                response = invoke_function(
                    "addNewBranch", [author, repo_name, branch_chaincode.json()]
                )
            except ChaincodeExecutionError as e:
                print("Deleting branch due to execution error", file=sys.stderr)
                delete_branch(repo_name, branch_name, repo_parent_directory)
                raise ChaincodeExecutionError(e)
        case "deleteBranch":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            repo_parent_directory = get_arg_at_position(other_args, 3, os.getcwd())

            try:
                response = invoke_function("deleteBranch", other_args[0:3])
                branch_chaincode = delete_branch(
                    repo_name, branch_name, repo_parent_directory
                )
            except ChaincodeExecutionError as e:
                raise ChaincodeExecutionError(e)
        case "checkout":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            repo_parent_directory = get_arg_at_position(other_args, 3, os.getcwd())

            checkout_branch(repo_name, branch_name, repo_parent_directory)
            response = f"Checked out branch {branch_name}"
        case "pull":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            repo_parent_directory = get_arg_at_position(other_args, 3, os.getcwd())

            response = invoke_function("queryBranch", other_args[0:3])
            branch_from_blockchain = Branch.model_validate(json.loads(response))
            blockchain_branch_commit_hashes = (
                get_all_commit_hashes_for_branch_from_chaincode(branch_from_blockchain)
            )
            local_branch_commit_hashes = get_all_commit_hashes_for_local_branch(
                repo_name, branch_name, repo_parent_directory
            )

            # Get latest commit hash that exists in both the blockchain and the local branch
            latest_common_commit_hash = get_latest_common_commit_hash(
                local_branch_commit_hashes, blockchain_branch_commit_hashes
            )

            response = invoke_function(
                "pull", [author, repo_name, branch_name, latest_common_commit_hash]
            )
            later_commits = [
                Commit.model_validate(element) for element in json.loads(response)
            ]
            if later_commits:
                add_pulled_commits_to_branch(
                    repo_name, branch_name, later_commits, repo_parent_directory
                )
            else:
                response = "Your branch is up to date"
        case "commit":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            commit_message = other_args[3]
            also_add_unstaged_files = other_args[4].lower() == "true"
            repo_parent_directory = get_arg_at_position(other_args, 5, os.getcwd())

            last_commit_hash_from_blockchain = invoke_function(
                "checkoutLast", other_args[0:3]
            )
            last_commit_from_repo = get_latest_commit_hash(
                repo_name, branch_name, repo_parent_directory
            )
            if last_commit_hash_from_blockchain != last_commit_from_repo:
                raise ValueError(
                    "Branch is not up to date with the info obtained from blockchain! Please pull the branch before committing."
                )

            commit = commit(
                repo_name,
                branch_name,
                commit_message,
                also_add_unstaged_files,
                repo_parent_directory,
            )
            # If a commit is generated, push it to the blockchain
            if commit is not None:
                try:
                    response = invoke_function(
                        "push", [author, repo_name, branch_name, commit.json()]
                    )
                except ChaincodeExecutionError as e:
                    print("Reverting commit due to execution error", file=sys.stderr)
                    revert_commit(
                        repo_name,
                        branch_name,
                        last_commit_from_repo,
                        repo_parent_directory,
                    )
                    raise ChaincodeExecutionError(e)
            else:
                response = "Working tree clean. Nothing to commit."
        case "push":
            author = other_args[0]
            repo_name = other_args[1]
            branch_name = other_args[2]
            repo_parent_directory = get_arg_at_position(other_args, 3, os.getcwd())

            last_commit_hash_from_blockchain = invoke_function(
                "checkoutLast", other_args[0:3]
            )
            local_branch_commit_hashes = get_all_commit_hashes_for_local_branch(
                repo_name, branch_name, repo_parent_directory
            )
            if last_commit_hash_from_blockchain not in local_branch_commit_hashes:
                raise ValueError(
                    "Last commit from blockchain is not part of local branch. Please pull the repo before attempting to commit."
                )

            commits_to_push = commits_after_specified_to_chaincode_structure(
                repo_name,
                branch_name,
                last_commit_hash_from_blockchain,
                repo_parent_directory,
            )

            # If there exist commits later than what is on the blockchain, push them to the blockchain
            if commits_to_push:
                try:
                    response = invoke_function(
                        "pushMultiple",
                        [
                            author,
                            repo_name,
                            branch_name,
                            json.dumps([json.loads(c.json()) for c in commits_to_push]),
                        ],
                    )
                except ChaincodeExecutionError as e:
                    print("Reverting commits due to execution error", file=sys.stderr)
                    # Reset branch to the last commit from blockchain
                    revert_commit(
                        repo_name,
                        branch_name,
                        last_commit_hash_from_blockchain,
                        repo_parent_directory,
                    )
                    raise ChaincodeExecutionError(e)
            else:
                response = "No commits to push."

        case "updateRepoAccess":
            # Arguments: repo.author repo.name authorizer_name user_access
            author = other_args[0]
            repo_name = other_args[1]
            user_to_authorize = other_args[2]
            access_value = UserAccess[other_args[3]].value

            response = invoke_function(
                "updateRepoUserAccess",
                [author, repo_name, user_to_authorize, access_value],
            )
        case "queryRepoAccess":
            # Arguments: repo.author repo.name
            response = invoke_function("queryRepoUserAccess", other_args)
        case _:
            raise NotImplementedError("Function not supported!")

    print(response)
