import js from "@eslint/js";
import stylistic from "@stylistic/eslint-plugin";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config(
  {
    ignores: ["dist/**", "node_modules/**"],
  },
  js.configs.recommended,
  {
    files: ["**/*.cjs"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "commonjs",
      globals: {
        ...globals.node,
      },
    },
  },
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    plugins: {
      "@stylistic": stylistic,
    },
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        ...globals.browser,
      },
    },
    rules: {
      "@stylistic/array-bracket-spacing": ["error", "never"],
      "@stylistic/comma-dangle": ["error", "always-multiline"],
      "@stylistic/indent": ["error", 2],
      "@stylistic/jsx-closing-bracket-location": ["error", "tag-aligned"],
      "@stylistic/jsx-closing-tag-location": "error",
      "@stylistic/jsx-curly-spacing": ["error", "never"],
      "@stylistic/jsx-equals-spacing": ["error", "never"],
      "@stylistic/jsx-first-prop-new-line": ["error", "multiline-multiprop"],
      "@stylistic/jsx-indent-props": ["error", 2],
      "@stylistic/jsx-max-props-per-line": ["error", { "maximum": 1, "when": "multiline" }],
      "@stylistic/jsx-tag-spacing": ["error", {
        "beforeSelfClosing": "always",
        "closingSlash": "never",
        "beforeClosing": "never",
        "afterOpening": "never"
      }],
      "@stylistic/object-curly-spacing": ["error", "always"],
      "max-len": ["warn", {
        "code": 100,
        "ignoreUrls": true,
        "ignoreStrings": true,
        "ignoreTemplateLiterals": true
      }],
      "no-empty": ["error", { "allowEmptyCatch": true }],
    },
  },
);
