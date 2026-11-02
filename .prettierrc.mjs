/**
 * @see https://prettier.io/docs/en/configuration.html
 * @type {import("prettier").Config}
 */
const config = {
  // prettier-plugin-tailwindcss-canonical-classes has to come after prettier-plugin-tailwindcss,
  // since it wraps the parser of the preceding plugin.
  plugins: [
    'prettier-plugin-organize-imports',
    'prettier-plugin-go-template',
    'prettier-plugin-tailwindcss',
    'prettier-plugin-tailwindcss-canonical-classes',
  ],
  tailwindStylesheet: './frontend/ui/src/styles/tailwind.css',
  tailwindcssCanonicalStylesheet: './frontend/ui/src/styles/tailwind.css',
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
