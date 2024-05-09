#!/usr/bin/env python3

import asyncio
import os.path

from hfc.fabric import Client

loop = asyncio.get_event_loop()

cli = Client(net_profile=f"{os.path.dirname(__file__)}/network.json")
cli.new_channel("mychannel")

org_config = {
    "org1": {
        "admin": cli.get_user(org_name="org1.example.com", name="Admin"),
        "user1": cli.get_user(org_name="org1.example.com", name="User1"),
        "peers": ["peer0.org1.example.com"],
    },
    "org2": {
        "admin": cli.get_user(org_name="org2.example.com", name="Admin"),
        "user1": cli.get_user(org_name="org2.example.com", name="User1"),
        "peers": ["peer0.org2.example.com"],
    },
}


def invoke_function(function_name: str, args: list[str]) -> str:
    """
    Invoke a function on the blockchain

    :param function_name: Name of function to invoke
    :param args: Arguments to pass in for the function called
    :return: Response obtained from blockchain
    """
    response = loop.run_until_complete(
        cli.chaincode_invoke(
            requestor=org_config["org2"]["user1"],
            channel_name="mychannel",
            peers=org_config["org2"]["peers"],
            fcn=function_name,
            args=args,
            cc_name="contract",
            transient_map=None,
            wait_for_event=True,
            raise_on_error=True,
        )
    )

    return response
