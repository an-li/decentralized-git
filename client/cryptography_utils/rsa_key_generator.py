#!/usr/bin/env python3

import os

from cryptography.hazmat.backends import default_backend
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa


def generate_rsa_key_pair() -> tuple[str, str]:
    """
    Generates a private and public keypair using RSA, then saves the private key to a pem file

    :param private_key_filepath: File path to save private key
    :return: private and public key expressed as strings
    """
    # Generate a private key
    private_key = rsa.generate_private_key(
        public_exponent=65537,  # Standard public exponent for RSA
        key_size=2048,  # Key size, you can adjust this as needed
        backend=default_backend(),
    )

    # Get the public key from the private key
    public_key = private_key.public_key()

    # Serialize the private key to PEM format
    private_pem = extract_private_key_bytes(private_key)

    # Serialize the public key to PEM format
    public_pem = extract_public_key_bytes(public_key)

    return private_pem.decode("utf-8"), public_pem.decode("utf-8")


def extract_private_key_bytes(private_key: rsa.RSAPrivateKey) -> bytes:
    return private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.TraditionalOpenSSL,
        encryption_algorithm=serialization.NoEncryption(),
    )


def extract_public_key_bytes(public_key: rsa.RSAPublicKey) -> bytes:
    return public_key.public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo,
    )


def read_rsa_private_key_from_pem(file_path: str) -> rsa.RSAPrivateKey:
    """Read the private key from the PEM file

    :param file_path: File path to read the key from
    :return: private key
    """
    with open(file_path, "rb") as f:
        private_key_pem = f.read()

    # Deserialize the private key from PEM format
    private_key: rsa.RSAPrivateKey = serialization.load_pem_private_key(
        private_key_pem,
        password=None,  # No password protection
        backend=default_backend(),
    )

    return private_key
