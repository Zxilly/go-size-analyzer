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
        'plugin:import/recommended',
        'plugin:import/typescript'
    ],
    ignorePatterns: [
        'dist',
        '.eslintrc.cjs',
        'coverage',
        'src/generated/schema.ts',
        'src/tool/wasm_exec.js',
    ],
    parser: '@typescript-eslint/parser',
    plugins: [
        'react-refresh',
    ],
    rules: {
        'sort-imports': ['warn', {ignoreDeclarationSort: true}],
        'import/no-unresolved': 'off',
        'react-refresh/only-export-components': [
            'warn',
            {allowConstantExport: true},
        ],
        '@typescript-eslint/no-unsafe-call': 'off',
        '@typescript-eslint/no-unsafe-return': 'off',
    },
    parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: ['./tsconfig.json', './tsconfig.node.json'],
        tsconfigRootDir: __dirname,
    },
    settings: {
        react: {
            version: 'detect',
        },
    },
}
