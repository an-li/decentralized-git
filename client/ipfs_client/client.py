#!/usr/bin/env python3

import shutil

import ipfshttpclient

# Connect to IPFS
ipfs = ipfshttpclient.connect("/ip4/127.0.0.1/tcp/5001")


def upload_to_ipfs(file_path: str) -> str:
    """
    Upload a file to IPFS given its file path

    :param file_path: File path
    :return: Hash of file added to IPFS
    """
    added_file_info = ipfs.add(file_path)
    return added_file_info._raw["Hash"]


def download_from_ipfs(ipfs_hash: str, destination_path: str):
    """
    Download a file from IPFS given its hash

    :param ipfs_hash: Hash of file in IPFS
    :param destination_path: Destination path to download file to
    :return:
    """
    ipfs.get(ipfs_hash)
    shutil.move(ipfs_hash, destination_path)
