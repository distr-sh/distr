// Writes a ZIP archive (https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT) of deflated
// regular files. Every entry gets the same timestamp, so the same files always produce the same bytes.

import {crc32, deflateRawSync} from 'node:zlib';

const VERSION = 20;
const MADE_BY_UNIX = (3 << 8) | VERSION;
const UTF8_NAMES = 1 << 11;
const DEFLATE = 8;
const DOS_TIME = 0;
const DOS_DATE = ((2020 - 1980) << 9) | (1 << 5) | 1;
const REGULAR_FILE_0644 = (0o100644 << 16) >>> 0;

export function zip(files) {
  const local = [];
  const central = [];
  let offset = 0;

  for (const [path, content] of Object.entries(files)) {
    const name = Buffer.from(path);
    const data = deflateRawSync(content);
    const crc = crc32(content);

    const header = Buffer.alloc(30);
    header.writeUInt32LE(0x04034b50, 0);
    header.writeUInt16LE(VERSION, 4);
    header.writeUInt16LE(UTF8_NAMES, 6);
    header.writeUInt16LE(DEFLATE, 8);
    header.writeUInt16LE(DOS_TIME, 10);
    header.writeUInt16LE(DOS_DATE, 12);
    header.writeUInt32LE(crc, 14);
    header.writeUInt32LE(data.length, 18);
    header.writeUInt32LE(content.length, 22);
    header.writeUInt16LE(name.length, 26);
    local.push(header, name, data);

    const entry = Buffer.alloc(46);
    entry.writeUInt32LE(0x02014b50, 0);
    entry.writeUInt16LE(MADE_BY_UNIX, 4);
    entry.writeUInt16LE(VERSION, 6);
    entry.writeUInt16LE(UTF8_NAMES, 8);
    entry.writeUInt16LE(DEFLATE, 10);
    entry.writeUInt16LE(DOS_TIME, 12);
    entry.writeUInt16LE(DOS_DATE, 14);
    entry.writeUInt32LE(crc, 16);
    entry.writeUInt32LE(data.length, 20);
    entry.writeUInt32LE(content.length, 24);
    entry.writeUInt16LE(name.length, 28);
    entry.writeUInt32LE(REGULAR_FILE_0644, 38);
    entry.writeUInt32LE(offset, 42);
    central.push(entry, name);

    offset += header.length + name.length + data.length;
  }

  const centralSize = central.reduce((size, part) => size + part.length, 0);
  const count = Object.keys(files).length;
  const end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50, 0);
  end.writeUInt16LE(count, 8);
  end.writeUInt16LE(count, 10);
  end.writeUInt32LE(centralSize, 12);
  end.writeUInt32LE(offset, 16);

  return Buffer.concat([...local, ...central, end]);
}
