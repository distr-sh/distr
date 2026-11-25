import {formatTolerationLines, parseKeyValueLines, parseTolerationLines} from './pod-scheduling';

describe('parseKeyValueLines', () => {
  it('should split each line at the first equals sign and skip blank lines', () => {
    expect(parseKeyValueLines('example.com/a = x=y\n\n  b=  \n')).toEqual({'example.com/a': 'x=y', b: ''});
  });

  it('should reject a line without a key', () => {
    expect(() => parseKeyValueLines('novalue')).toThrow();
    expect(() => parseKeyValueLines('=value')).toThrow();
  });
});

describe('parseTolerationLines', () => {
  it('should read the taint syntax of kubectl', () => {
    expect(parseTolerationLines('CriticalAddonsOnly\ndedicated=ctl:NoSchedule\n:NoExecute')).toEqual([
      {key: 'CriticalAddonsOnly', operator: 'Exists', effect: undefined},
      {key: 'dedicated', operator: 'Equal', value: 'ctl', effect: 'NoSchedule'},
      {key: undefined, operator: 'Exists', effect: 'NoExecute'},
    ]);
  });

  it('should reject an unknown effect and a value without a key', () => {
    expect(() => parseTolerationLines('a:NoRun')).toThrow();
    expect(() => parseTolerationLines('=b')).toThrow();
  });

  it('should format what it parses back to the same text', () => {
    const text = 'CriticalAddonsOnly:NoSchedule\ndedicated=ctl\n:NoExecute';
    expect(formatTolerationLines(parseTolerationLines(text))).toEqual(text);
  });
});
