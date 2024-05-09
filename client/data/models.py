#!/usr/bin/env python3

from datetime import datetime
from enum import Enum

from pydantic import BaseModel


class UserAccess(Enum):
    ReadAccess = 1
    ReadWriteAccess = 2
    OwnerAccess = 3
    NoAccess = 4


class Commit(BaseModel):
    hash: str
    author: str
    authorEmail: str
    message: str
    parentHashes: list[str]
    timestamp: datetime
    storageHashes: dict[str, str]


class CommitWithBranch(Commit):
    branch: str


class Push(BaseModel):
    branchName: str
    commits: list[Commit]


class Branch(BaseModel):
    name: str
    commits: dict[str, Commit]


class AccessLog(BaseModel):
    authorizer: str
    authorized: str
    timestamp: datetime
    userAccess: UserAccess


class Repository(BaseModel):
    name: str
    author: str
    directoryCID: str
    commitHashes: dict[str, bool]
    access: dict[str, UserAccess]
    branches: dict[str, Branch]
    accessLogs: list[AccessLog]
