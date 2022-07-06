import { readFileSync } from 'fs'
import { parse, join, resolve, relative, dirname } from 'path'
import walkSync from 'walk-sync'
import { unified } from 'unified'
import remarkParse from 'remark-parse'
import remarkMdx from 'remark-mdx'
import remarkRehype from 'remark-rehype'
import { visit } from 'unist-util-visit'
import { toString } from 'hast-util-to-string'
import Slugger from 'github-slugger'
import { closest } from 'fastest-levenshtein'

const pathPrefix = '/cadence/'
const urlPrefix = 'https://docs.onflow.org/'

function findPaths() {
    return walkSync('.', {
        directories: false,
        globs: ['**/*.md', '**/*.mdx'],
        ignore: ['**/node_modules']
    })
}

const processor = unified()
    .use(remarkParse)
    .use(remarkMdx)
    .use(remarkRehype)

function parseDocument(path) {
    const contents = readFileSync(path).toString()
    try {
        return processor.runSync(processor.parse(contents))
    } catch (e) {
        console.error(`failed to parse ${path}`)
        throw e
    }
}

const slugger = new Slugger()

const targets = {}

function basename(path) {
    const {name, dir} = parse(path)
    if (name === 'index') {
        return dir
    }
    return join(dir, name)
}

function index(document, path) {

    slugger.reset()

    visit(document, 'element', (node) => {
        if (!['h2', 'h3'].includes(node.tagName)) {
            return
        }

        let id = node.properties.id
        if (!id) {
            id = slugger.slug(toString(node))
        }

        const target = basename(path)
        let ids = targets[target]
        if (!ids) {
            ids = new Set()
            targets[target] = ids
        }
        // TODO: check if exists
        ids.add(id.toLowerCase())
    })
}

function check(document, path) {

    function warn(node, message, suggestion) {
        const { href } = node.properties
        const { line, column } = node.position.start
        suggestion = suggestion ? `. ${suggestion}` : ''
        console.warn(`${path}:${line}:${column}: ${message}: ${href}${suggestion}`)
    }

    const checkedTarget = basename(path)

    visit(document, 'element', (node) => {
        if (node.tagName !== 'a') {
            return
        }

        let { href } = node.properties

        if (href.startsWith(urlPrefix)) {
            warn(node, "link includes domain")
            return
        }

        if (href.startsWith('http:')) {
            warn(node, "link with insecure HTTP URL")
            return
        }

        if (href.startsWith('https:')) {
            return
        }

        let [linkedTarget, linkedID] = href.split('#')
        if (linkedTarget === '') {
            linkedTarget = checkedTarget
        } else {
            if (linkedTarget.match(/^\.\.?/)) {
                warn(node, "relative link")
                return
            }

            if (linkedTarget.match(/\.mdx?$/)) {
                warn(node, "link with extension")
                return
            }

            linkedTarget = linkedTarget.replace(/\/$/, "")

            if (linkedTarget.startsWith(pathPrefix)) {
                linkedTarget = linkedTarget.substr(pathPrefix.length)
            } else {
                if (linkedTarget.startsWith('/language')) {
                    warn(node, "suspicious absolute link")
                    return
                }

                if (linkedTarget.startsWith('/')) {
                    return
                }

                linkedTarget = relative('.', resolve(dirname(path), linkedTarget))
            }
        }

        const ids = targets[linkedTarget]
        if (!ids) {
            warn(node, "unknown link target")
            return
        }

        if (linkedID && !ids.has(linkedID.toLowerCase())) {
            let suggestion = ''
            if (ids.size > 0) {
                suggestion = `did you mean #${closest(linkedID, Array.from(ids))}?`
            }

            warn(node, "unknown link target ID", suggestion)
            return
        }
    })
}

const paths = findPaths()
const documents = new Map(paths.map(path =>
    [path, parseDocument(path)]
))

documents.forEach(index)

documents.forEach(check)
