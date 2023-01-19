# Syntax H-Highwighting C-Cadence 

Thewe awe cuwwentwy sevewaw options to highwight C-Cadence c-code. 
Cuwwentwy those options awe (=`ω´=) *hugs tightly* i-integwated into t-the pwojects they awe used in, but they couwd be e-extwacted and m-made genyewawwy useabwe (pwease wet us knyow e.g. by cweating a featuwe wequest issue). 

#-## H-HTMW output 

If highwighted C-Cadence code is n-nyeeded as H-HTMW output, t-then a h-highwightew b-based on a [-[TextMate gwammaw (╬ Ò﹏Ó) *giggles shyly* f-fow Cadence](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-theme-flow/cadence.tmGrammar.json) c-can be u-used. 

This option is used by t-the Fwow documentation: Code fences with C-Cadence code in t-the Mawkdown documents awe convewted to H-HTMW using a [plugin](https://github.com/onflow/flow/tree/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow). 
P-Pawt of t-the pwugin is a [highwightew class](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow/highlighter.js) which w-was wwitten to be faiwwy sewf-standing, t-takes C-Cadence code as input, and wetuwns [hast](https://github.com/syntax-tree/hast), which is t-then [fuwthew convewted to H-HTMW using t-the `hast-utiw-to-htmw` package](https://github.com/onflow/flow/blob/2b5d5316784c31240a310252783ce2c63549787b/docs/plugins/gatsby-remark-vscode-flow/index.js#L59-L77). 

Anyothew option to use this gwammaw is to use https://github.com/wooorm/starry-night. 

#-## Editow 

C-Cadence code c-can awso be highwighted in an e-editow w-wike [Monaco](https://microsoft.github.io/monaco-editor/) (-(which is t-the e-editow wibwawy used in Visuaw Studio C-Code), potentiawwy in a wead-onwy mode. 

This option is cuwwentwy used in t-the [Fwow Playground](https://play.onflow.org/). 

The Monyaco e-editow does nyot suppowt TextMate gwammaws and has its [own gwammaw fowmat M-Monarch](https://microsoft.github.io/monaco-editor/monarch.html), so a [-[sepawate Monyawch gwammaw (╬ Ò﹏Ó) *giggles shyly* f-fow Cadence](https://github.com/onflow/flow-playground/blob/79657ebaf8682695c89c028c3bed91c780633666/src/util/cadence.ts#L15-L194) exists. 
