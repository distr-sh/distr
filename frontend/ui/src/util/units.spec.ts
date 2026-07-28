import {formatBytes} from './units';

describe('formatBytes', () => {
  it('should format 1000 to 1,000B', () => expect(formatBytes(1000, 'en-US')).toEqual('1,000B'));
  it('should format 1200 to 1.172KiB', () => expect(formatBytes(1200, 'en-US')).toEqual('1.172KiB'));
  it('should format -1024 to -1KiB', () => expect(formatBytes(-1024, 'en-US')).toEqual('-1KiB'));
  it('should format 8734568 to 8.33MiB', () => expect(formatBytes(8734568, 'en-US')).toEqual('8.33MiB'));
  it('should format 1.5TiB to 1.5TiB', () => expect(formatBytes(1649267441664, 'en-US')).toEqual('1.5TiB'));
  it('should format 2PiB to 2,048TiB', () => expect(formatBytes(2251799813685248, 'en-US')).toEqual('2,048TiB'));
  it('should format 0 to 0B', () => expect(formatBytes(0, 'en-US')).toEqual('0B'));
});
