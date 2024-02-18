import argparse
from Cryptodome.Cipher import AES
import hashlib
import io
import struct
import sys
import time
from typing import BinaryIO
import zlib


XMLC = b'XMLC'
Y2K = 0x20000301


def genkeys(hdr: bytes) -> tuple[bytes]:
    magic, epoch = struct.unpack_from('<4sI', hdr)

    if magic != XMLC and epoch != Y2K:
        raise ValueError('Bad XMLC header')

    # aes-key, iv
    return (hashlib.md5(b'\x12\x34\x56\x78' + hdr).digest() + hashlib.md5(b'\x0a\xbc\xd7\x75').digest(),
            hashlib.md5(b'\x91\x2a\x54\xb7').digest())


def encrypt(in_file: BinaryIO, out_file: BinaryIO) -> None:
    key, iv = genkeys(in_file.read(32))

    cipher = AES.new(key, AES.MODE_OFB, iv=iv)

    out_file.write(cipher.encrypt(in_file.read()))


def decompress(in_file: BinaryIO, out_file: BinaryIO) -> None:
    with io.BytesIO() as out_buf:
        encrypt(in_file, out_buf)

        out_file.write(zlib.decompress(out_buf.getbuffer(), wbits=31))


def compress(in_file: BinaryIO, out_file: BinaryIO) -> None:
    hdr = struct.pack('<4s2I20x', XMLC, Y2K, int(time.time()))

    out_file.write(hdr)

    with io.BytesIO() as in_buf:
        in_buf.write(hdr)
        in_buf.write(zlib.compress(in_file.read(), wbits=31))
        in_buf.seek(0)

        encrypt(in_buf, out_file)


def main() -> None:
    parser = argparse.ArgumentParser(description='XMLC tool')
    parser.add_argument('-c', '--compress', action='store_true')
    parser.add_argument("infile",
                    type=argparse.FileType('rb'))
    parser.add_argument("outfile",
                    type=argparse.FileType('wb'))

    args = parser.parse_args()

    with args.infile as in_file, args.outfile as out_file:
        action = compress if args.compress else decompress

        try:
            action(in_file, out_file)
        except (ValueError, zlib.error) as e:
            sys.exit(str(e))

    sys.exit(0)


if __name__ == '__main__':
    main()
