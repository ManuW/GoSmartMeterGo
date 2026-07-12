#!/usr/bin/env python3

import socket
import struct

# Speedwire
MULTICAST_GROUP = "239.12.255.254"
MULTICAST_PORT = 9522

def printFrame(tag, data):
    if tag == 0x0000:
        print("TAG: End-of-Data")
    elif tag == 0x02a0:
        print("TAG: Tag0 (42), version 0")
        if data == b"\00\00\00\01":
            print("    Group1 (default group)")
        else:
            print(f"    Unknown group: {data.hex()}")
    elif tag == 0x0010:
        print("TAG: SMA Net 2, version 0")

    else:
        print(f"Unknown tag: {tag}")


sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("", MULTICAST_PORT))

mreq = struct.pack("4sl", socket.inet_aton(MULTICAST_GROUP), socket.INADDR_ANY)
sock.setsockopt(socket.IPPROTO_IP, socket.IP_ADD_MEMBERSHIP, mreq)

while True:
    data, addr = sock.recvfrom(1024)

    print(f"Received data from {addr}, length: {len(data)}")

    # Hex-Dump der Daten
    # print(f"\nData: {data.hex()}\n")

    startSeq = data[0:4]
    if startSeq == b"SMA\0":
        print(f"Start Sequence: {startSeq}")

        idx = 4
        while idx < len(data):
            if idx + 4 > len(data):
                print(f"Index out of bounds: {idx}")
                break
            jdx = idx + 2
            length = int.from_bytes(data[idx:jdx], byteorder="big", signed=False)
            idx = jdx
            jdx = idx + 2
            tag = int.from_bytes(data[idx:jdx], byteorder="big", signed=False)
            idx = jdx
            jdx = idx + length
            if jdx > len(data):
                print(f"Index out of bounds: {jdx}")
                break
            payload = data[idx:jdx]
            idx = jdx
            jdx = idx + 2
            # print(f"Tag: {tag}, Length: {length}, Payload: {payload.hex()}")
            printFrame(tag, payload)

    else:
        print(f"Unexpected start sequence: {startSeq}")

    print("")
