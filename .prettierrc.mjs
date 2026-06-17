/**
 * @see https://prettier.io/docs/en/configuration.html
 * @type {import("prettier").Config}
 */
const config = {
  plugins: ['prettier-plugin-organize-imports', 'prettier-plugin-go-template'],
  overrides: [
    {
      files: ['internal/**/*.html'],
      options: {
        parser: 'go-template',
      },
    },
  ],
  bracketSameLine: true,
  bracketSpacing: false,
  printWidth: 120,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};

export default config;
