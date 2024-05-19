/** @type { import("eslint").Linter.BaseConfig } */
module.exports = {
    root: true,
    env: {browser: true, es2020: true},
    extends: [
        'eslint:recommended',
        'plugin:react/recommended',
        'plugin:react/jsx-runtime',
        'plugin:@typescript-eslint/recommended-type-checked',
        'plugin:react-hooks/recommended',
    ],
    ignorePatterns: [
        'dist',
        '.eslintrc.cjs',
        'src/generated/schema.ts'
    ],
    parser: '@typescript-eslint/parser',
    plugins: [
        'react-refresh',
        'eslint-plugin-react-compiler',
    ],
    rules: {
        'react-refresh/only-export-components': [
            'warn',
            {allowConstantExport: true},
        ],
        '@typescript-eslint/no-unsafe-call': 'off',
        '@typescript-eslint/no-unsafe-return': 'off',
        'react-compiler/react-compiler': "error",
    },
    parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: ['./tsconfig.json', './tsconfig.node.json'],
        tsconfigRootDir: __dirname,
    },
}
