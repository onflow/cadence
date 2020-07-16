import * as monaco from "monaco-editor";

export const CADENCE_LANGUAGE_ID = "cadence"

interface CadenceMonarchLanguage extends monaco.languages.IMonarchLanguage {
}

export function configureCadence() {

  monaco.languages.register({
    id: CADENCE_LANGUAGE_ID,
    extensions: [".cdc"],
    aliases: ["CDC", "cdc"]
  });

  monaco.languages.setMonarchTokensProvider(CADENCE_LANGUAGE_ID, {

    keywords: [
      "if",
      "else",
      "return",
      "continue",
      "break",
      "while",
      "pre",
      "post",
      "prepare",
      "execute",
      "import",
      "from",
      "create",
      "destroy",
      "priv",
      "pub",
      "get",
      "set",
      "log",
      "emit",
      "event",
      "init",
      "struct",
      "interface",
      "fun",
      "let",
      "var",
      "resource",
      "access",
      "all",
      "contract",
      "self",
      "transaction"
    ],

    typeKeywords: [
      "AnyStruct",
      "AnyResource",
      "Void",
      "Never",
      "String",
      "Character",
      "Bool",
      "Self",
      "Int8",
      "Int16",
      "Int32",
      "Int64",
      "Int128",
      "Int256",
      "UInt8",
      "UInt16",
      "UInt32",
      "UInt64",
      "UInt128",
      "UInt256",
      "Word8",
      "Word16",
      "Word32",
      "Word64",
      "Word128",
      "Word256",
      "Fix64",
      "UFix64"
    ],

    operators: [
      "<-",
      "<=",
      ">=",
      "==",
      "!=",
      "+",
      "-",
      "*",
      "/",
      "%",
      "&",
      "!",
      "&&",
      "||",
      "?",
      "??",
      ":",
      "=",
      "@"
    ],

    // we include these common regular expressions
    symbols: /[=><!~?:&|+\-*\/\^%]+/,
    escapes: /\\(?:[abfnrtv\\"']|x[0-9A-Fa-f]{1,4}|u[0-9A-Fa-f]{4}|U[0-9A-Fa-f]{8})/,
    digits: /\d+(_+\d+)*/,
    octaldigits: /[0-7]+(_+[0-7]+)*/,
    binarydigits: /[0-1]+(_+[0-1]+)*/,
    hexdigits: /[[0-9a-fA-F]+(_+[0-9a-fA-F]+)*/,
    tokenizer: {
      root: [[/[{}]/, "delimiter.bracket"], {include: "common"}],

      common: [
        // identifiers and keywords
        [
          /[a-z_$][\w$]*/,
          {
            cases: {
              "@typeKeywords": "keyword",
              "@keywords": "keyword",
              "@default": "identifier"
            }
          }
        ],
        [/[A-Z][\w\$]*/, "type.identifier"], // to show class names nicely
        // [/[A-Z][\w\$]*/, 'identifier'],

        // whitespace
        {include: "@whitespace"},

        // delimiters and operators
        [/[()\[\]]/, "@brackets"],
        [/[<>](?!@symbols)/, "@brackets"],
        [
          /@symbols/,
          {
            cases: {
              "@operators": "delimiter",
              "@default": ""
            }
          }
        ],

        // numbers
        [/(@digits)[eE]([\-+]?(@digits))?/, "number.float"],
        [/(@digits)\.(@digits)([eE][\-+]?(@digits))?/, "number.float"],
        [/0[xX](@hexdigits)/, "number.hex"],
        [/0[oO]?(@octaldigits)/, "number.octal"],
        [/0[bB](@binarydigits)/, "number.binary"],
        [/(@digits)/, "number"],

        // delimiter: after number because of .\d floats
        [/[;,.]/, "delimiter"],

        // strings
        [/"([^"\\]|\\.)*$/, "string.invalid"], // non-teminated string
        [/'([^'\\]|\\.)*$/, "string.invalid"], // non-teminated string
        [/"/, "string", "@string_double"],
        [/'/, "string", "@string_single"]
      ],

      whitespace: [
        [/[ \t\r\n]+/, ""],
        [/\/\*/, "comment", "@comment"],
        [/\/\/.*$/, "comment"]
      ],

      comment: [
        [/[^\/*]+/, "comment"],
        [/\*\//, "comment", "@pop"],
        [/[\/*]/, "comment"]
      ],
      string_double: [
        [/[^\\"]+/, "string"],
        [/@escapes/, "string.escape"],
        [/\\./, "string.escape.invalid"],
        [/"/, "string", "@pop"]
      ],
      string_single: [
        [/[^\\']+/, "string"],
        [/@escapes/, "string.escape"],
        [/\\./, "string.escape.invalid"],
        [/'/, "string", "@pop"]
      ]
    }
  } as CadenceMonarchLanguage);
}
