import {slugMaxLength, slugPattern, toSlug} from './slug';

describe('toSlug', () => {
  const cases: [name: string, expected: string][] = [
    ['Acme SSO', 'acme-sso'],
    ["Acme's  SSO", 'acme-s-sso'],
    ['  Entra ID  ', 'entra-id'],
    ['Öko GmbH – Login', 'ko-gmbh-login'],
    ['--dashes--', 'dashes'],
    ['Okta (EU) 2.0', 'okta-eu-2-0'],
  ];

  for (const [name, expected] of cases) {
    it(`should derive ${JSON.stringify(expected)} from ${JSON.stringify(name)}`, () => {
      const slug = toSlug(name);
      expect(slug).toEqual(expected);
      // The derived slug is submitted unchanged, so it also has to satisfy what the API validates.
      expect(slug).toMatch(slugPattern);
    });
  }

  it('should truncate a name that exceeds the maximum length', () => {
    expect(toSlug('a'.repeat(slugMaxLength + 10))).toEqual('a'.repeat(slugMaxLength));
  });

  it('should strip the trailing hyphen that truncation leaves behind', () => {
    expect(toSlug(`${'a'.repeat(slugMaxLength - 1)} b`)).toEqual('a'.repeat(slugMaxLength - 1));
  });

  it('should return an empty slug for a name without alphanumerics', () => {
    expect(toSlug('--- ---')).toEqual('');
  });
});
