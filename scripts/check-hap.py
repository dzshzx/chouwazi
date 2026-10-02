#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
#
# [tool.uv]
# exclude-newer = "3 days"
# ///
"""Validate a HAP against the independently compiled AOT code, even after strip."""
import hashlib
import json
from pathlib import Path
import re
import struct
import sys
import zipfile


def aot_sections(data):
    # The release target is little-endian ELF64 AArch64. Stripping changes the
    # symbol table, but must not change compiled code or snapshot data.
    header = struct.unpack_from('<16sHHIQQQIHHHHHH', data)
    assert data[:6] == b'\x7fELF\x02\x01' and header[2] == 183, 'Expected arm64 ELF'
    sections = [struct.unpack_from('<IIQQQQIIQQ', data, header[6] + i * header[11])
                for i in range(header[12])]
    strings = sections[header[13]]
    names = data[strings[4]:strings[4] + strings[5]]
    result = {}
    for section in sections:
        name = names[section[0]:].split(b'\0', 1)[0]
        if name in (b'.text', b'.rodata'):
            result[name.decode()] = hashlib.sha256(data[section[4]:section[4] + section[5]]).hexdigest()
    assert set(result) == {'.text', '.rodata'}, 'Missing Dart AOT code or snapshot data'
    return result


def main():
    hap_path, aot_path = map(Path, sys.argv[1:3])
    version = re.search(r'^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$', Path('pubspec.yaml').read_text(), re.M)
    assert version, 'Missing application version'
    with zipfile.ZipFile(hap_path) as hap:
        for name in ['module.json', 'pack.info', 'libs/arm64-v8a/libapp.so', 'libs/arm64-v8a/libflutter.so']:
            assert name in hap.namelist(), f'Missing {name}'
        manifest = json.loads(hap.read('module.json'))
        app = manifest['app']
        assert app['bundleName'] == 'com.alex.chouwazi'
        assert app['versionName'] == version[1] and app['versionCode'] == int(version[2])
        permissions = {p['name'] for p in manifest.get('module', {}).get('requestPermissions', [])}
        assert permissions <= {'ohos.permission.VIBRATE'}, f'Unexpected permissions: {permissions}'
        assert aot_sections(hap.read('libs/arm64-v8a/libapp.so')) == aot_sections(aot_path.read_bytes()), 'HAP contains stale AOT code'
    print(f'Verified {hap_path}: identity, version, permissions and current Dart AOT code.')


if __name__ == '__main__':
    main()
