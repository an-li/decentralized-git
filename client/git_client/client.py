#!/usr/bin/env python3

import os
from datetime import datetime

import git
import pytz
from git import Actor, GitCommandError, Head, Repo

from data.models import Branch, Commit, CommitWithBranch, Repository
from ipfs_client.client import download_from_ipfs, upload_to_ipfs


def set_git_user_config(name: str, email: str):
    """
    Set system Git user name email to match logged on user's information

    :param name: Name of logged on user
    :param email: Email of logged on user
    :return:
    """
    os.system(f'git config --global user.name "{name}"')
    os.system(f'git config --global user.email "{email}"')


def initialize_repo(repo_location: str, author_name: str, author_email: str) -> Repo:
    """
    Initialize a new repo at repo_location, then set the Git config for it with the user's name and email

    :param repo_location: Path at which the repo is located
    :param author_name: Logged in user's name
    :param author_email: Logged in user's email
    :return: Git repo
    """
    repo = Repo.init(repo_location, initial_branch="main")

    repo.index.commit(
        message="Initial commit",
        author=Actor(author_name, author_email),
        committer=Actor(author_name, author_email),
    )

    repo.git.checkout("main")

    return repo


def initialize_repo_from_chaincode_structure(
    repository: Repository,
    repo_parent_directory: str,
    branch_to_checkout: str = "main",
):
    """
    Initialize repo read from blockchain, then set the Git config for it with the user's name and email

    :param checkoutter_name: Logged in user's name
    :param checkoutter_email: Logged in user's email
    :param repository: Repo obtained from blockchain
    :param repo_parent_directory: Directory where the repo is located
    :param branch_to_checkout: Branch to checkout (default = main)
    :return: Git repo
    """
    commits_in_chronological_order = get_all_commits_in_chronological_order(repository)
    repo = Repo.init(
        f"{repo_parent_directory}/{repository.name}",
        initial_branch=commits_in_chronological_order[0].branch,
    )

    # Change working directory to where the repo is located
    os.chdir(repo.working_dir)

    for commit in commits_in_chronological_order:
        if repo.heads:
            # Checkout branch only if there is at least one commit
            current = (
                repo.branches[commit.branch]
                if commit.branch in repo.heads
                else repo.create_head(commit.branch)
            )
            current.checkout()

        # Commit does not exist, need to create new one
        for filename, storage_hash in commit.storageHashes.items():
            download_from_ipfs(storage_hash, filename)
        repo.git.add(A=True)
        repo.index.commit(
            message=commit.message,
            parent_commits=[repo.commit(h) for h in commit.parentHashes],
            author=Actor(commit.author, commit.authorEmail),
            committer=Actor(commit.author, commit.authorEmail),
            author_date=commit.timestamp,
            commit_date=commit.timestamp,
        )
    repo.git.checkout(branch_to_checkout)

    # Return to where the client is located
    os.chdir(os.path.dirname(os.path.dirname(__file__)))


def add_pulled_commits_to_branch(
    repo_name: str,
    branch_name: str,
    commits: list[Commit],
    repo_parent_directory: str,
):
    """
    Add commits pulled from blockchain to local git instance

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param commits: List of commits obtained from blockchain
    :param repo_parent_directory: Parent directory where the repo is located
    :return:
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)

    # Change working directory to where the repo is located
    os.chdir(repo.working_dir)

    for commit in sorted(commits, key=lambda x: x.timestamp):
        # Go through commits from oldest to newest
        try:
            repo.git.checkout(commit.hash)
        except GitCommandError:
            # Commit does not exist, need to create new one
            for filename, storage_hash in commit.storageHashes.items():
                download_from_ipfs(storage_hash, filename)
            repo.git.add(A=True)
            repo.index.commit(
                message=commit.message,
                parent_commits=[repo.commit(h) for h in commit.parentHashes],
                author=Actor(commit.author, commit.authorEmail),
                committer=Actor(commit.author, commit.authorEmail),
                author_date=commit.timestamp,
                commit_date=commit.timestamp,
            )

    # Return to where the client is located
    os.chdir(os.path.dirname(os.path.dirname(__file__)))


def create_new_branch(
    repo_name: str, branch_name: str, repo_parent_directory: str
) -> Branch:
    """
    Create new branch and marshal it to chaincode format

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param repo_parent_directory: Parent directory where the repo is located
    :return: New branch created
    """
    if branch_name == "main":
        raise ValueError("Cannot create main branch again!")

    repo = Repo(f"{repo_parent_directory}/{repo_name}")
    try:
        # Attempt to get the branch reference
        repo.git.rev_parse("--verify", branch_name)
        raise ValueError(f"Cannot create branch {branch_name} more than once!")
    except:
        # If the branch reference does not exist, an exception will be raised
        pass

    branch = repo.create_head(branch_name)
    branch.checkout()

    branch = branch_to_chaincode_structure(repo, branch)

    return branch


def checkout_branch(
    repo_name: str, branch_name: str, repo_parent_directory: str
) -> Repo:
    """
    Checkout specified branch in the repo and return its state

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param repo_parent_directory: Parent directory where the repo is located
    :return: Repo with specified branch checked out and active
    """
    repo = Repo(f"{repo_parent_directory}/{repo_name}")
    repo.git.checkout(branch_name)
    return repo


def get_all_commit_hashes_for_branch_from_chaincode(branch: Branch) -> list[str]:
    """
    Get all commit hashes for the specified branch from chaincode

    :param branch: Branch obtained from blockchain
    :return: All commit hashes for the specified branch
    """
    return [
        h
        for h, commit in sorted(
            branch.commits.items(), key=lambda item: item[1].timestamp
        )
    ]


def get_all_commits_in_chronological_order(repo: Repository) -> list[CommitWithBranch]:
    """
    Get all commits for repo coming from blockchain in chronologincal order

    :param repo: Repository object coming from blockchain
    :return: Commits sorted in chronological order
    """
    commits = [
        CommitWithBranch(
            hash=c.hash,
            author=c.author,
            authorEmail=c.authorEmail,
            message=c.message,
            parentHashes=c.parentHashes,
            timestamp=c.timestamp,
            storageHashes=c.storageHashes,
            branch=b.name,
        )
        for b in repo.branches.values()
        for c in b.commits.values()
    ]

    return sorted(commits, key=lambda c: c.timestamp)


def get_all_commit_hashes_for_local_branch(
    repo_name: str, branch_name: str, repo_parent_directory: str
) -> list[str]:
    """
    Get all commit hashes for the specified local branch

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param repo_parent_directory: Parent directory where the repo is located
    :return: All commit hashes for the specified branch
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)

    # Get the currently checked out branch
    current_branch = repo.active_branch
    commits = [commit.hexsha for commit in repo.iter_commits(current_branch)]
    commits.reverse()
    return commits


def get_latest_common_commit_hash(
    hashes_from_local: list[str], hashes_from_blockchain: list[str]
) -> str | None:
    """
    Return the latest common commit hash on the local branch that exists on the blockchain

    :param hashes_from_local: Commit hashes from local branch
    :param hashes_from_blockchain: Commit hashes for branch in blockchain
    :return: Commit hash of latest local commit that also exists in blockchain, or None if none of the local commits are part of blockchain
    """
    # Sort hashes in reverse order (newest to oldest)
    hashes_from_local.reverse()
    hashes_from_blockchain.reverse()

    return next(
        iter([h for h in hashes_from_local if h in hashes_from_blockchain]),
        None,
    )


def get_latest_commit_hash(
    repo_name: str, branch_name: str, repo_parent_directory: str
):
    """
    Checkout branch and get its latest commit hash

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param repo_parent_directory: Parent directory where the repo is located
    :return: Latest commit hash
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)

    # Get the currently checked out branch
    current_branch = repo.active_branch

    # Get the last commit of the current branch
    last_commit = current_branch.commit

    return last_commit.hexsha


def delete_branch(
    repo_name: str,
    branch_name: str,
    repo_parent_directory: str,
    branch_to_checkout_after_deletion: str = "main",
):
    """
    Delete branch, then checkout specified branch after its deletion

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param repo_parent_directory: Parent directory where the repo is located
    :param branch_to_checkout_after_deletion: Branch to check out after deletion
    """
    if branch_name == "main":
        raise ValueError("main branch cannot be deleted!")

    repo = checkout_branch(
        repo_name, branch_to_checkout_after_deletion, repo_parent_directory
    )
    repo.delete_head(branch_name)


def commit(
    repo_name: str,
    branch_name: str,
    commit_message: str,
    also_add_untracked_files: bool,
    repo_parent_directory: str,
) -> Commit:
    """
    Commit staged and unstaged files currently in repository and push
    If the working tree is clean, return and cancel

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param commit_message: Commit message
    :param repo_parent_directory: Parent directory where the repo is located
    :param also_add_untracked_files: If True, also add untracked files to commit
    :return: Commit to be pushed to blockchain, None if working tree clean
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)

    # Change working directory to where the repo is located
    os.chdir(repo.working_dir)

    if not repo.is_dirty(untracked_files=also_add_untracked_files):
        # Working tree is clean, nothing to commit
        # Return to where the client is located
        os.chdir(os.path.dirname(os.path.dirname(__file__)))
        return None

    if also_add_untracked_files:
        repo.git.add(A=True)
    commit = repo.index.commit(message=commit_message)
    commit_cc = commit_to_chaincode_structure(repo, commit)

    repo.git.checkout(branch_name)

    # Return to where the client is located
    os.chdir(os.path.dirname(os.path.dirname(__file__)))

    return commit_cc


def commits_after_specified_to_chaincode_structure(
    repo_name: str, branch_name: str, commit_hexsha: str, repo_parent_directory: str
) -> list[Commit]:
    """
    Convert commits after specified one into the structure to be pushed into chaincode

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param commit_hexsha: SHA-1 hast of latest commit obtained from blockchain
    :param repo_parent_directory: Parent directory where the repo is located
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)
    specified_commit = repo.commit(commit_hexsha)

    # Change working directory to where the repo is located
    os.chdir(repo.working_dir)

    commits = [
        commit_to_chaincode_structure(repo, commit)
        for commit in repo.iter_commits(branch_name)
        if commit.committed_datetime > specified_commit.committed_datetime
    ]
    repo.git.checkout(branch_name)

    # Return to where the client is located
    os.chdir(os.path.dirname(os.path.dirname(__file__)))

    # Reverse commits so they are from oldest to newest
    commits.reverse()

    return commits


def revert_commit(
    repo_name: str,
    branch_name: str,
    commit_sha: str,
    repo_parent_directory: str,
):
    """
    Revert commit with given SHA

    :param repo_name: Name of Git repo
    :param branch_name: Name of branch
    :param commit_sha: SHA of commit to revert
    :param repo_parent_directory: Parent directory where the repo is located
    """
    repo = checkout_branch(repo_name, branch_name, repo_parent_directory)

    # Reset the branch to the specified commit
    repo.git.reset("--hard", commit_sha)


def convert_repository_to_chaincode_structure(
    author: str, repo_name: str, repo_parent_directory: str
) -> Repository:
    """
    Convert Git repo into the structure to be pushed into chaincode

    :param author: Name of repo's author
    :param repo_name: Name of Git repo
    :param repo_parent_directory: Parent directory where the repo is located
    """
    repo = Repo(f"{repo_parent_directory}/{repo_name}")

    # Change working directory to where the repo is located
    os.chdir(repo.working_dir)

    branches = {
        head.name: branch_to_chaincode_structure(repo, head) for head in repo.branches
    }

    commit_hashes = {
        c.hash: True for b in branches.values() for c in b.commits.values()
    }

    # Return to where the client is located
    os.chdir(os.path.dirname(os.path.dirname(__file__)))

    return Repository(
        name=repo_name,
        author=author,
        directoryCID=repo_name,
        commitHashes=commit_hashes,
        branches=branches,
        access={},
        accessLogs=[],
    )


def branch_to_chaincode_structure(repo: Repo, branch: Head) -> Branch:
    """
    Convert Git branch into the structure to be pushed into chaincode

    :param repo: Git repo
    :param branch: Git branch
    """
    current = repo.create_head(branch.name)
    current.checkout()
    commits = {
        commit.hexsha: commit_to_chaincode_structure(repo, commit)
        for commit in repo.iter_commits(branch)
    }
    return Branch(name=branch.name, commits=commits)


def commit_to_chaincode_structure(repo: Repo, commit: git.Commit):
    """
    Convert Git commit into the structure to be pushed into chaincode

    :param repo: Git repo
    :param commit: Git commit
    """
    repo.git.checkout(commit.hexsha)
    parent_hashes = [c.hexsha for c in commit.parents]
    storage_hashes = {f: upload_to_ipfs(f) for f in commit.stats.files.keys()}
    cc_commit = Commit(
        hash=commit.hexsha,
        author=commit.author.name,
        authorEmail=commit.author.email,
        message=commit.message,
        parentHashes=parent_hashes,
        timestamp=commit.committed_datetime,
        storageHashes=storage_hashes,
    )
    return cc_commit
